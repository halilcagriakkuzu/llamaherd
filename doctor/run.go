package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"llamaherd/settings"
)

// Run executes every check in a sensible order and summarizes the result.
func Run() Report {
	r := Report{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Platform:    platformName(),
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	r.Checks = append(r.Checks,
		checkServerBin(),
		checkServerVersion(),
		checkGPU(),
		checkGUILibs(),
		checkDisplay(),
		checkModelsDir(),
		checkWSLModels(),
		checkDisk(),
		checkAPIPort(),
	)
	r.OK, r.Summary = summarize(r.Checks)
	return r
}

// summarize counts outcomes (ok is true when nothing failed).
func summarize(checks []Check) (ok bool, summary string) {
	var fails, warns int
	for _, c := range checks {
		switch c.Status {
		case StatusFail:
			fails++
		case StatusWarn:
			warns++
		}
	}
	ok = fails == 0
	if fails == 0 && warns == 0 {
		return ok, "all checks passed"
	}
	var parts []string
	if fails > 0 {
		parts = append(parts, plural(fails, "failure"))
	}
	if warns > 0 {
		parts = append(parts, plural(warns, "warning"))
	}
	return ok, strings.Join(parts, ", ")
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// platformName is a human label ("WSL (Linux)", "macOS", ...).
func platformName() string {
	switch runtime.GOOS {
	case "linux":
		if settings.IsWSL() {
			return "WSL (Linux)"
		}
		return "Linux"
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	}
	return runtime.GOOS
}

// execOut runs a command with a timeout and returns trimmed stdout+stderr.
func execOut(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
