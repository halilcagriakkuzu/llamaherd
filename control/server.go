// Package control is the ServerManager layer (task 11): real process
// lifecycle for llama-server. UI-agnostic — the Wails app wires its
// callbacks into runtime events. Real I/O (net/http, os/exec) lives here,
// per the README phase-2 rules.
package control

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"llamaherd/schema"
)

// Status mirrors the README state machine.
// stopped -> starting -> running -> stopped, plus crashed.
type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusStopping Status = "stopping"
	StatusRunning  Status = "running"
	StatusCrashed  Status = "crashed"
)

// Info is the manager's exportable state snapshot.
type Info struct {
	Status   Status
	Exit     int
	PID      int
	External bool // running, but started outside LlamaHerd — Stop refuses
	ReadyAt  time.Time
	Listen   string
	VRAMMB   int // live GPU memory of the process (0 = unknown)
	RAMMB    int // live RSS of the process
	NCtx     int // real n_ctx from /props (0 = unknown)
}

// Manager controls one llama-server child at a time.
type Manager struct {
	bin      string
	mu       sync.Mutex
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	status   Status
	exit     int
	external bool
	readyAt  time.Time
	listen   string
	vramMB   int
	ramMB    int
	nCtx     int
	onChange func() // called on every state transition (any goroutine)
	onLog    func(string)
}

// NewManager creates the manager for a llama-server binary path.
func NewManager(bin string) *Manager {
	return &Manager{bin: bin, status: StatusStopped}
}

// SetCallbacks wires state-change and log callbacks. Both may be called from
// goroutines.
func (m *Manager) SetCallbacks(onChange func(), onLog func(string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = onChange
	m.onLog = onLog
}

func (m *Manager) log(line string) {
	m.mu.Lock()
	f := m.onLog
	m.mu.Unlock()
	if f != nil {
		f(fmt.Sprintf("server: %s", line))
	}
}

func (m *Manager) emit() {
	m.mu.Lock()
	f := m.onChange
	m.mu.Unlock()
	if f != nil {
		f()
	}
}

// Info snapshots the current state.
func (m *Manager) Info() Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	info := Info{
		Status:   m.status,
		Exit:     m.exit,
		External: m.external,
		ReadyAt:  m.readyAt,
		Listen:   m.listen,
		VRAMMB:   m.vramMB,
		RAMMB:    m.ramMB,
		NCtx:     m.nCtx,
	}
	if m.cmd != nil && m.cmd.Process != nil {
		info.PID = m.cmd.Process.Pid
	}
	return info
}

// AttachExternal checks whether something already serves /health on
// host:port; if yes the manager enters `running (external)` so the UI shows a
// live LlamaHerd did not spawn.
func (m *Manager) AttachExternal(host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	if !healthOK(addr, 2*time.Second) {
		return
	}
	m.mu.Lock()
	m.status = StatusRunning
	m.external = true
	m.listen = addr
	m.readyAt = time.Now() // uptime counts from attach
	m.mu.Unlock()
	m.log(fmt.Sprintf("existing server detected on %s (started outside LlamaHerd; Stop refuses until you Start here)", addr))
	m.emit()
}

func (m *Manager) setStatusLocked(s Status) { m.status = s }

// Start launches llama-server with the given config.
func (m *Manager) Start(cfg schema.Config) error {
	m.mu.Lock()
	if m.status != StatusStopped && m.status != StatusCrashed {
		m.mu.Unlock()
		return fmt.Errorf("already %s", m.status)
	}
	addr := fmt.Sprintf("%s:%d", cfg.Basic.Host, cfg.Basic.Port)
	if conflictsPort(addr) {
		m.mu.Unlock()
		return fmt.Errorf("something is already listening on %s — Stop it first", addr)
	}
	name, argv := m.bin, BuildArgs(cfg)
	if w, ok := ParseWSLBin(m.bin); ok {
		if runtime.GOOS != "windows" {
			m.mu.Unlock()
			return fmt.Errorf("%q is only valid when LlamaHerd runs on Windows", m.bin)
		}
		name, argv = w.Command(argv)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, name, argv...)
	cmd.Cancel = func() error { return terminate(cmd) }
	cmd.WaitDelay = 5 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		m.mu.Unlock()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		m.mu.Unlock()
		return err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		m.mu.Unlock()
		return err
	}
	m.cancel = cancel
	m.cmd = cmd
	m.external = false
	m.exit = 0
	m.listen = addr
	m.setStatusLocked(StatusStarting)
	pid := cmd.Process.Pid
	startedAt := time.Now()
	m.mu.Unlock()

	m.log(fmt.Sprintf("pid %d spawned: %s %s", pid, name, strings.Join(argv, " ")))
	m.emit()

	go m.pipeLogs(stdout, "stdout")
	go m.pipeLogs(stderr, "stderr")
	go m.awaitExit(cmd, ctx)
	go m.awaitHealth(ctx, cfg.Basic.Host, cfg.Basic.Port, startedAt)
	return nil
}

// Stop terminates the managed process: SIGTERM (via Cancel) then the
// WaitDelay escalates to SIGKILL after 3s.
func (m *Manager) Stop() error {
	m.mu.Lock()
	if m.status == StatusStopped {
		m.mu.Unlock()
		return nil
	}
	if m.external {
		m.mu.Unlock()
		return errors.New("refusing to Stop: server was started outside LlamaHerd")
	}
	if m.cancel == nil {
		m.status = StatusStopped
		m.mu.Unlock()
		return nil
	}
	if m.status == StatusRunning || m.status == StatusStarting {
		m.status = StatusStopping
	}
	cancel := m.cancel
	listen := m.listen
	pid := 0
	if m.cmd != nil {
		pid = m.cmd.Process.Pid
	}
	m.mu.Unlock()

	m.log(fmt.Sprintf("sending SIGTERM (pid %d)", pid))
	m.emit()
	cancel()
	if w, ok := ParseWSLBin(m.bin); ok {
		// Killing wsl.exe does not reliably stop the Linux process.
		if _, port, ok := splitHostPort(listen); ok {
			name, args := w.StopCommand(port)
			go func() { _ = exec.Command(name, args...).Run() }()
		}
	}
	return nil
}

// pipeLogs forwards one output stream to the log callback.
func (m *Manager) pipeLogs(r interface{ Read([]byte) (int, error) }, stream string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		m.log(fmt.Sprintf("[%s] %s", stream, sc.Text()))
	}
}

// awaitExit records the process outcome.
func (m *Manager) awaitExit(cmd *exec.Cmd, ctx context.Context) {
	err := cmd.Wait()
	wasCancelled := ctx.Err() != nil
	m.mu.Lock()
	m.cancel = nil
	m.cmd = nil
	m.vramMB = 0
	m.ramMB = 0
	m.nCtx = 0
	var line string
	switch {
	case wasCancelled:
		m.status = StatusStopped
		line = "stopped"
	default:
		m.exit = exitCode(err)
		m.status = StatusCrashed
		line = fmt.Sprintf("crashed: exit status %d", m.exit)
		m.readyAt = time.Time{}
	}
	// Collect the callback + message under the lock, but invoke the callback
	// OUTSIDE it: m.log() itself locks m.mu (non-reentrant), so calling it
	// while holding the lock self-deadlocks and freezes the whole manager.
	f := m.onLog
	m.mu.Unlock()

	if f != nil {
		f("server: " + line)
	}
	m.emit()
}

// awaitHealth polls /health until the server answers (flip to running) or
// gives up after 60s, logging but leaving the state machine untouched.
func (m *Manager) awaitHealth(ctx context.Context, host string, port int, startedAt time.Time) {
	addr := fmt.Sprintf("%s:%d", host, port)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
		m.mu.Lock()
		running := m.status == StatusStarting
		m.mu.Unlock()
		if !running {
			return // exited or transitioned elsewhere
		}
		if healthOK(addr, 750*time.Millisecond) {
			m.mu.Lock()
			if m.status == StatusStarting {
				m.status = StatusRunning
				m.readyAt = time.Now()
			}
			ready := m.status == StatusRunning
			stoppedAlready := m.status == StatusStopped || m.status == StatusCrashed
			m.mu.Unlock()
			if ready {
				el := time.Since(startedAt).Truncate(100 * time.Millisecond)
				m.log(fmt.Sprintf("ready in %s (%s)", el, addr))
				go m.sampleMetrics(ctx)
				m.emit()
			}
			if stoppedAlready {
				return
			}
			return
		}
	}
	m.log("server not healthy after 60s — still waiting (check LOGS)")
}

// sampleMetrics polls VRAM/RSS every 5s while the process lives; n_ctx is
// fetched once on the first tick (it doesn't change over the process life).
func (m *Manager) sampleMetrics(ctx context.Context) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	ctxDone := ctx.Done()
	for i := 0; ; i++ {
		m.mu.Lock()
		pid := 0
		addr := m.listen
		running := m.status == StatusRunning && m.cmd != nil && m.cmd.Process != nil
		if running {
			pid = m.cmd.Process.Pid
		}
		m.mu.Unlock()
		if !running {
			return
		}
		vram := VramMB(pid)
		ram := RamMB(pid)
		nc := 0
		if i == 0 {
			nc = NCtx(addr)
		}
		m.mu.Lock()
		m.vramMB = vram
		m.ramMB = ram
		if i == 0 && nc > 0 {
			m.nCtx = nc
		}
		m.mu.Unlock()
		m.emit() // metrics-only refresh even without log lines
		select {
		case <-ctxDone:
			return
		case <-tick.C:
		}
	}
}

// healthOK answers whether /health responds 200.
func healthOK(addr string, timeout time.Duration) bool {
	cl := &http.Client{Timeout: timeout}
	resp, err := cl.Get("http://" + addr + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// conflictsPort is the README launch guard: something already listening.
func conflictsPort(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false // no listener (or dial failed in a way that means "no")
	}
	conn.Close()
	return true
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

// BuildArgs returns the llama-server arguments for a config, in the flag
// order used by the user's reference commands. Shares the assembly rules
// documented in task 07 (skip empty; skip spec-* unless a draft type; booleans
// only when checked; 0 = skip for numerics).
func BuildArgs(cfg schema.Config) []string {
	b, adv := cfg.Basic, cfg.Advanced
	var args []string
	add := func(flag, val string) {
		if val != "" {
			args = append(args, flag, val)
		}
	}
	add("--model", b.Model)
	if b.Alias != "" {
		add("--alias", b.Alias)
	}
	add("--host", b.Host)
	if b.Port > 0 {
		add("--port", fmt.Sprint(b.Port))
	}
	add("-ngl", b.GPULayers)
	add("--fit", adv.Fit)
	if b.Context > 0 {
		add("-c", fmt.Sprint(b.Context))
	}
	if b.ParallelSlots > 0 {
		add("-np", fmt.Sprint(b.ParallelSlots))
	}
	add("-fa", b.FlashAttention)
	add("-ctk", b.KCache)
	add("-ctv", b.VCache)
	if b.Jinja {
		args = append(args, "--jinja")
	}
	add("--reasoning-format", b.ReasoningFmt)
	if strings.HasPrefix(adv.SpecType, "draft-") {
		add("--spec-type", adv.SpecType)
		// -md/--spec-draft-model is a separate draft sidecar; draft-mtp reads
		// its heads from the target model, so DraftModel stays empty there.
		add("--spec-draft-model", adv.DraftModel)
		if adv.DraftNMax > 0 {
			add("--spec-draft-n-max", fmt.Sprint(adv.DraftNMax))
		}
		add("--spec-draft-ngl", adv.DraftNgl)
		add("--spec-draft-type-k", adv.DraftTypeK)
		add("--spec-draft-type-v", adv.DraftTypeV)
	}
	add("--mmproj", adv.MMProj)
	if adv.NoMMProjOffload {
		args = append(args, "--no-mmproj-offload")
	}
	if adv.Perf {
		args = append(args, "--perf")
	}
	if adv.SleepIdle > 0 {
		add("--sleep-idle-seconds", fmt.Sprint(adv.SleepIdle))
	}
	if adv.CacheRAM > 0 {
		add("--cache-ram", fmt.Sprint(adv.CacheRAM))
	}
	if adv.Mlock {
		args = append(args, "-mlock")
	}
	if adv.NoMmap {
		args = append(args, "-no-mmap")
	}
	if adv.CPUThreads > 0 {
		add("-t", fmt.Sprint(adv.CPUThreads))
	}
	if adv.Batch > 0 {
		add("-b", fmt.Sprint(adv.Batch))
	}
	if adv.UBatch > 0 {
		add("-ub", fmt.Sprint(adv.UBatch))
	}
	add("-ot", adv.OverrideTensor)
	return args
}

// splitHostPort parses "host:port".
func splitHostPort(addr string) (string, int, bool) {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return "", 0, false
	}
	n, err := strconv.Atoi(addr[i+1:])
	return addr[:i], n, err == nil
}
