//go:build darwin

package control

import (
	"os/exec"
	"strconv"
	"strings"
)

// serverProc is one llama-server process discovered on this machine.
type serverProc struct {
	pid    int
	args   []string
	uptime string
}

// listServerProcs finds llama-server processes via pgrep/ps (macOS has no
// /proc filesystem).
func listServerProcs() []serverProc {
	out, err := exec.Command("pgrep", "-x", "llama-server").Output()
	if err != nil {
		return nil
	}
	var procs []serverProc
	for _, line := range strings.Fields(string(out)) {
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		args := strings.Fields(psField(pid, "args"))
		procs = append(procs, serverProc{pid: pid, args: args, uptime: psField(pid, "etime")})
	}
	return procs
}

// psField runs `ps -o <field>= -p <pid>` and returns the trimmed value.
func psField(pid int, field string) string {
	out, err := exec.Command("ps", "-o", field+"=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// procRSS reports pid's resident set size in MiB, 0 when unknown.
func procRSS(pid int) int {
	kb, err := strconv.Atoi(psField(pid, "rss"))
	if err != nil {
		return 0
	}
	return kb / 1024
}

// TotalRAMMB reads total physical RAM in MiB via sysctl, 0 when unknown.
func TotalRAMMB() int {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	b, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return int(b / (1024 * 1024))
}
