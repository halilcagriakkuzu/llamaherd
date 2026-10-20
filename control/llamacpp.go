// llamacpp.go — information about the local llama.cpp checkout/build, the
// llama-server processes on the machine (ours or anyone else's), a force-stop
// for them, and the runner for scripts/update-llamacpp.sh.
package control

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"llamaherd/settings"
)

// LlamaInfo describes the checkout and its build. Fields are strings so the
// UI can show them as-is; Err is set when the checkout cannot be read.
type LlamaInfo struct {
	RepoDir    string `json:"repoDir"`
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`     // short hash
	Describe   string `json:"describe"`   // git describe (tag-based build number)
	CommitDate string `json:"commitDate"` // author date of HEAD
	Subject    string `json:"subject"`
	Dirty      bool   `json:"dirty"`   // tracked files modified
	Behind     int    `json:"behind"`  // commits behind upstream as of the last fetch; -1 unknown
	Fetched    string `json:"fetched"` // when upstream was last fetched ("" = never)
	Binary     string `json:"binary"`
	BuiltAt    string `json:"builtAt"`
	Version    string `json:"version"` // `llama-server --version`
	BuildDir   string `json:"buildDir"`
	BuildType  string `json:"buildType"`
	CudaOn     bool   `json:"cudaOn"`
	CudaArch   string `json:"cudaArch"`
	GPU        string `json:"gpu"`
	Err        string `json:"err"`
}

// LlamaProc is one running llama-server process.
type LlamaProc struct {
	PID     int    `json:"pid"`
	Uptime  string `json:"uptime"`
	Listen  string `json:"listen"` // from --host/--port, "" if absent
	Model   string `json:"model"`
	Cmd     string `json:"cmd"`
	Managed bool   `json:"managed"` // spawned by this app
}

var (
	verMu   sync.Mutex
	verMtim time.Time
	verText string
)

func git(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// ReadLlamaInfo gathers checkout/build/GPU facts. It never fetches.
func ReadLlamaInfo(repo, binary string) LlamaInfo {
	li := LlamaInfo{RepoDir: repo, Binary: binary, BuildDir: settings.LlamaBuildDir(), Behind: -1}
	var err error
	if li.Commit, err = git(repo, "rev-parse", "--short", "HEAD"); err != nil {
		li.Err = "git checkout unreadable: " + repo
		return li
	}
	li.Branch, _ = git(repo, "rev-parse", "--abbrev-ref", "HEAD")
	li.Describe, _ = git(repo, "describe", "--tags", "--always")
	li.CommitDate, _ = git(repo, "log", "-1", "--format=%ad", "--date=format:%Y-%m-%d %H:%M")
	li.Subject, _ = git(repo, "log", "-1", "--format=%s")
	if out, _ := git(repo, "status", "--porcelain", "--untracked-files=no"); out != "" {
		li.Dirty = true
	}
	if out, e := git(repo, "rev-list", "--count", "HEAD..@{u}"); e == nil {
		if n, e := strconv.Atoi(out); e == nil {
			li.Behind = n
		}
	}

	if st, err := os.Stat(repo + "/.git/FETCH_HEAD"); err == nil {
		li.Fetched = st.ModTime().Format("2006-01-02 15:04")
	}

	if st, err := os.Stat(binary); err == nil {
		li.BuiltAt = st.ModTime().Format("2006-01-02 15:04")
		li.Version = binaryVersion(binary, st.ModTime())
	} else {
		li.Version = "binary missing"
	}

	if f, err := os.Open(repo + "/" + settings.LlamaBuildDir() + "/CMakeCache.txt"); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			k, v, ok := strings.Cut(sc.Text(), "=")
			if !ok {
				continue
			}
			switch {
			case strings.HasPrefix(k, "CMAKE_BUILD_TYPE:"):
				li.BuildType = v
			case strings.HasPrefix(k, "GGML_CUDA:"):
				li.CudaOn = v == "ON"
			case strings.HasPrefix(k, "CMAKE_CUDA_ARCHITECTURES:"):
				li.CudaArch = v
			}
		}
	}
	li.GPU = gpuName()
	return li
}

func binaryVersion(bin string, mtime time.Time) string {
	verMu.Lock()
	defer verMu.Unlock()
	if verMtim.Equal(mtime) && verText != "" {
		return verText
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, _ := exec.CommandContext(ctx, bin, "--version").CombinedOutput()
	line := ""
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "version:") {
			line = strings.TrimSpace(strings.TrimPrefix(l, "version:"))
		}
	}
	if line == "" {
		line = "unknown"
	}
	verMtim, verText = mtime, line
	return line
}

func gpuName() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=name,compute_cap,memory.total", "--format=csv,noheader").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
}

// FetchLlama runs `git fetch` so Behind becomes meaningful.
func FetchLlama(repo string) error {
	_, err := git(repo, "fetch", "--quiet")
	return err
}

// ListLlamaProcs returns the llama-server processes found on this machine (any
// origin). managedPID marks the one this app spawned (0 = none). Discovery is
// platform-specific (proc_linux.go / proc_darwin.go / proc_windows.go).
func ListLlamaProcs(managedPID int) []LlamaProc {
	var procs []LlamaProc
	for _, sp := range listServerProcs() {
		p := LlamaProc{
			PID:     sp.pid,
			Cmd:     strings.Join(sp.args, " "),
			Managed: sp.pid == managedPID,
			Uptime:  sp.uptime,
		}
		host, port := "127.0.0.1", ""
		for i := 0; i+1 < len(sp.args); i++ {
			switch sp.args[i] {
			case "--host":
				host = sp.args[i+1]
			case "--port":
				port = sp.args[i+1]
			case "-m", "--model":
				p.Model = sp.args[i+1][strings.LastIndexAny(sp.args[i+1], "/\\")+1:]
			}
		}
		if port != "" {
			p.Listen = host + ":" + port
		}
		procs = append(procs, p)
	}
	return procs
}

// KillLlamaProcs force-stops every llama-server process, whoever started it:
// a graceful stop first (SIGTERM on Unix, TerminateProcess on Windows), then a
// forced kill for whatever is still alive after the grace period. Returns the
// PIDs that were signalled.
func KillLlamaProcs(grace time.Duration) []int {
	procs := ListLlamaProcs(0)
	var pids []int
	for _, p := range procs {
		if signalGraceful(p.PID) == nil {
			pids = append(pids, p.PID)
		}
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) && len(ListLlamaProcs(0)) > 0 {
		time.Sleep(200 * time.Millisecond)
	}
	for _, p := range ListLlamaProcs(0) {
		_ = forceKill(p.PID)
	}
	return pids
}

// ResetExternal clears the "running (external)" / stale state after the
// process behind it was force-killed.
func (m *Manager) ResetExternal() {
	m.mu.Lock()
	if m.external || m.cmd == nil {
		m.external = false
		m.status = StatusStopped
		m.listen = ""
		m.readyAt = time.Time{}
		m.vramMB, m.ramMB, m.nCtx = 0, 0, 0
	}
	m.mu.Unlock()
	m.emit()
}

// RunUpdateScript runs scripts/update-llamacpp.sh, forwarding every output
// line to logf. cancel() terminates the whole process group (cmake/ninja
// children included).
func RunUpdateScript(script string, logf func(string)) (cancel func(), done <-chan error) {
	ch := make(chan error, 1)
	cmd := exec.Command("bash", script)
	cmd.SysProcAttr = groupAttr()
	pr, pw, _ := os.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		ch <- err
		return func() {}, ch
	}
	pw.Close()
	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			logf("[llama.cpp] " + sc.Text())
		}
		pr.Close()
		ch <- cmd.Wait()
	}()
	return func() { killGroup(cmd.Process.Pid) }, ch
}

// ProbeBinary checks that a llama-server candidate actually runs and returns its
// version line. A "wsl:" path is probed inside the distro.
func ProbeBinary(bin string) (string, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return "", fmt.Errorf("empty llama-server path")
	}
	name, args := bin, []string{"--version"}
	if w, ok := ParseWSLBin(bin); ok {
		name, args = w.Run(w.Path, "--version")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", fmt.Errorf("%s --version failed: %w", bin, err)
		}
		return "", fmt.Errorf("%s --version failed: %w: %s", bin, err, firstLine(text))
	}
	if text == "" {
		return "", fmt.Errorf("%s --version produced no output", bin)
	}
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "version:") {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "version:")), nil
		}
	}
	return firstLine(text), nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
