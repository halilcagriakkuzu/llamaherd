//go:build unix && !linux && !darwin

package control

// Best-effort fallbacks for other Unix systems (the BSDs): process discovery
// and memory reporting are unavailable, but the app still builds and runs.

type serverProc struct {
	pid    int
	args   []string
	uptime string
}

func listServerProcs() []serverProc { return nil }
func procRSS(pid int) int           { return 0 }
func TotalRAMMB() int               { return 0 }
