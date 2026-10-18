package doctor

import (
	"fmt"
	"runtime"
	"strings"
)

// linuxLib reports a GTK/WebKit dependency version via pkg-config (dev
// headers) then ldconfig (runtime .so).
func linuxLib(pc, so string) (string, bool) {
	if v, err := execOut("pkg-config", "--modversion", pc); err == nil && v != "" {
		return v, true
	}
	if out, err := execOut("ldconfig", "-p"); err == nil && strings.Contains(out, so) {
		return "present", true
	}
	return "", false
}

const aptFix = "sudo apt-get update && sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev"

// checkGUILibs verifies the libraries the GUI needs on this OS.
func checkGUILibs() Check {
	c := Check{ID: "gui-libs", Title: "GUI libraries"}
	switch runtime.GOOS {
	case "linux":
		return linuxGUILibs(c)
	case "darwin":
		if p, err := execOut("xcode-select", "-p"); err == nil && p != "" {
			c.Status, c.Detail = StatusOK, "Command Line Tools: "+p
		} else {
			c.Status = StatusFail
			c.Detail = "Xcode Command Line Tools not found"
			c.Fix = "xcode-select --install"
		}
		return c
	case "windows":
		return webview2Check(c)
	}
	c.Status, c.Detail = StatusSkip, "not applicable on "+runtime.GOOS
	return c
}

func linuxGUILibs(c Check) Check {
	gtk, okG := linuxLib("gtk+-3.0", "libgtk-3.so")
	webkit, okW := linuxLib("webkit2gtk-4.1", "libwebkit2gtk-4.1.so")
	if !okW {
		webkit, okW = linuxLib("webkit2gtk-4.0", "libwebkit2gtk-4.0.so")
	}
	if okG && okW {
		c.Status = StatusOK
		c.Detail = fmt.Sprintf("GTK3 %s, WebKit2GTK %s", gtk, webkit)
		return c
	}
	c.Status = StatusFail
	c.Fix = aptFix
	switch {
	case !okG && !okW:
		c.Detail = "libgtk-3 and libwebkit2gtk not found"
	case !okG:
		c.Detail = "libgtk-3 not found (webkit2gtk " + webkit + " ok)"
	default:
		c.Detail = "libwebkit2gtk not found (gtk3 " + gtk + " ok)"
	}
	return c
}
