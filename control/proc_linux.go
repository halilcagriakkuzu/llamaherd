//go:build linux

package control

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// serverProc is one llama-server process discovered on this machine, in the
// small shape ListLlamaProcs needs (args, not a joined command line, so paths
// with spaces survive).
type serverProc struct {
	pid    int
	args   []string
	uptime string
}

// listServerProcs scans /proc for llama-server processes of any origin.
func listServerProcs() []serverProc {
	entries, _ := os.ReadDir("/proc")
	var out []serverProc
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		comm, _ := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
		if strings.TrimSpace(string(comm)) != "llama-server" {
			continue
		}
		raw, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		args := strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
		out = append(out, serverProc{pid: pid, args: args, uptime: procUptime(pid)})
	}
	return out
}

// procRSS reports the resident set size of pid in MiB (VmRSS), 0 when unknown.
func procRSS(pid int) int {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line[len("VmRSS:"):])
		if len(fields) >= 1 {
			if kb, err := strconv.Atoi(fields[0]); err == nil {
				return kb / 1024
			}
		}
	}
	return 0
}

// procUptime formats how long pid has been alive (from /proc/<pid> mtime).
func procUptime(pid int) string {
	st, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	if err != nil {
		return "—"
	}
	d := time.Since(st.ModTime()).Truncate(time.Second)
	return fmt.Sprintf("%02d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}

// TotalRAMMB reads total system RAM in MiB (MemTotal), 0 when unknown.
func TotalRAMMB() int {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		f := strings.Fields(line)
		if len(f) >= 2 {
			if kb, err := strconv.Atoi(f[1]); err == nil {
				return kb / 1024
			}
		}
	}
	return 0
}
