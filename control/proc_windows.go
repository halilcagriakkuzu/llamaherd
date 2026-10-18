//go:build windows

package control

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// serverProc is one llama-server process discovered on this machine.
type serverProc struct {
	pid    int
	args   []string
	uptime string
}

const (
	detachedProcess       = 0x00000008 // DETACHED_PROCESS
	createNewProcessGroup = 0x00000200 // CREATE_NEW_PROCESS_GROUP
)

// listServerProcs finds llama-server.exe processes via tasklist. Command lines
// are intentionally not retrieved (that needs WMI/`wmic`, which is being
// removed from Windows) so Model/Listen stay empty on Windows.
func listServerProcs() []serverProc {
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq llama-server.exe",
		"/FO", "CSV", "/NH").Output()
	if err != nil {
		return nil
	}
	var procs []serverProc
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "INFO:") {
			continue
		}
		// "llama-server.exe","1234","Console","1","1,234 K"
		fields := strings.Split(line, `","`)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(strings.Trim(fields[1], `"`))
		if err != nil {
			continue
		}
		procs = append(procs, serverProc{pid: pid, uptime: "—"})
	}
	return procs
}

// procRSS reports pid's working set in MiB via tasklist, 0 when unknown.
func procRSS(pid int) int {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid),
		"/FO", "CSV", "/NH").Output()
	if err != nil {
		return 0
	}
	line := strings.TrimSpace(string(out))
	if line == "" || strings.HasPrefix(line, "INFO:") {
		return 0
	}
	fields := strings.Split(line, `","`)
	kb := strings.ReplaceAll(strings.Trim(fields[len(fields)-1], `" `), ",", "")
	kb = strings.TrimSuffix(strings.TrimSpace(kb), " K")
	n, err := strconv.Atoi(strings.TrimSpace(kb))
	if err != nil {
		return 0
	}
	return n / 1024
}

// TotalRAMMB reads total physical RAM in MiB via kernel32, 0 when unknown.
func TotalRAMMB() int {
	k := syscall.NewLazyDLL("kernel32.dll")
	proc := k.NewProc("GetPhysicallyInstalledSystemMemory")
	var kb uint64
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&kb)))
	if r == 0 {
		return 0
	}
	return int(kb / 1024)
}

// terminate kills the child: Windows has no SIGTERM, so a hard kill is the
// only reliable stop (the exec.Cmd WaitDelay then reaps it).
func terminate(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

func signalGraceful(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

func forceKill(pid int) error { return signalGraceful(pid) }

// detachedAttr starts the child detached from the console so the headless
// daemon outlives the launching terminal.
func detachedAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: detachedProcess}
}

// groupAttr starts the child in its own process group.
func groupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
}

// killGroup kills the process group's leader (Windows has no group signal).
func killGroup(pid int) { _ = signalGraceful(pid) }
