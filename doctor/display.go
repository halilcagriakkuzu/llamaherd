package doctor

import (
	"os"
	"runtime"
)

// checkDisplay verifies a GUI display is available (Linux/WSL only).
func checkDisplay() Check {
	c := Check{ID: "display", Title: "Display / GUI session"}
	if runtime.GOOS != "linux" {
		c.Status, c.Detail = StatusSkip, "not applicable on "+runtime.GOOS
		return c
	}
	disp := os.Getenv("DISPLAY")
	_, xerr := os.Stat("/tmp/.X11-unix/X0")
	switch {
	case disp != "":
		c.Status, c.Detail = StatusOK, "DISPLAY="+disp
	case xerr == nil:
		c.Status = StatusWarn
		c.Detail = "DISPLAY is unset but an X socket exists at :0"
		c.Fix = "launch with: DISPLAY=:0 GSK_RENDERER=cairo llamaherd"
	default:
		c.Status = StatusWarn
		c.Detail = "no DISPLAY / X server found (headless)"
		c.Fix = "use the CLI/daemon (llamaherd daemon), or set DISPLAY and GSK_RENDERER=cairo"
	}
	return c
}
