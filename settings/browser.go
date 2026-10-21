package settings

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// browserLauncher is one way to hand a URL to the desktop. wait is true for
// launchers that return as soon as the browser was spawned (so their exit code
// tells us whether it worked); xdg-open / open are fire-and-forget because some
// implementations block until the browser exits.
type browserLauncher struct {
	argv []string
	wait bool
}

// OpenExternalURL opens raw in the user's default browser.
//
// Inside WSL the Linux side usually has no browser registered, so xdg-open
// succeeds at nothing: the URL is handed to the Windows host first (wslu's
// wslview when installed, then PowerShell, then cmd.exe). A launcher that
// starts but exits non-zero is treated as a failure, so a dead end becomes a
// real error instead of a silent no-op.
func OpenExternalURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty url")
	}
	var attempts []browserLauncher
	switch {
	case IsWSL():
		attempts = append(attempts,
			browserLauncher{[]string{"wslview", raw}, true},
			browserLauncher{[]string{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Start-Process '" + psQuote(raw) + "'"}, true},
			browserLauncher{[]string{"cmd.exe", "/c", "start", "", raw}, true},
			browserLauncher{[]string{"xdg-open", raw}, false},
		)
	case runtime.GOOS == "windows":
		attempts = append(attempts,
			browserLauncher{[]string{"rundll32", "url.dll,FileProtocolHandler", raw}, true},
			browserLauncher{[]string{"cmd", "/c", "start", "", raw}, true},
		)
	case runtime.GOOS == "darwin":
		attempts = append(attempts, browserLauncher{[]string{"open", raw}, false})
	default:
		attempts = append(attempts, browserLauncher{[]string{"xdg-open", raw}, false})
	}

	var lastErr error
	for _, l := range attempts {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c := exec.CommandContext(ctx, l.argv[0], l.argv[1:]...)
		if err := c.Start(); err != nil {
			cancel()
			lastErr = err
			continue
		}
		if !l.wait {
			go func(c *exec.Cmd, cancel context.CancelFunc) {
				_ = c.Wait()
				cancel()
			}(c, cancel)
			return nil
		}
		err := c.Wait()
		cancel()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("%s: %w", l.argv[0], err)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no browser launcher found")
	}
	return fmt.Errorf("could not open %s: %w", raw, lastErr)
}

// psQuote escapes a value for a single-quoted PowerShell string literal.
func psQuote(s string) string { return strings.ReplaceAll(s, "'", "''") }
