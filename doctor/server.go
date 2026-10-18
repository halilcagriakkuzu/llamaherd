package doctor

import (
	"os"
	"runtime"
	"strconv"

	"llamaherd/settings"
)

// MinLlamaBuild is the oldest llama.cpp build considered safe (it must know
// the flags the presets use, e.g. --spec-type draft-mtp, --cache-ram).
// Override with LLAMAHERD_MIN_LLAMA_BUILD.
const MinLlamaBuild = 4000

func minBuild() int {
	if v := os.Getenv("LLAMAHERD_MIN_LLAMA_BUILD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return MinLlamaBuild
}

func checkServerBin() Check {
	c := Check{ID: "server-bin", Title: "llama-server binary"}
	p := settings.ServerBin()
	st, err := os.Stat(p)
	switch {
	case err != nil:
		c.Status = StatusFail
		c.Detail = "not found at " + p
		c.Fix = "install llama.cpp, then set the path:\n" +
			"  LLAMAHERD_SERVER_BIN=/path/to/llama-server llamaherd doctor\n" +
			"  (or set \"serverBin\" in " + settings.ConfigFile() + ")"
	case st.IsDir():
		c.Status = StatusFail
		c.Detail = "is a directory, not a file: " + p
		c.Fix = "point LLAMAHERD_SERVER_BIN at the llama-server executable"
	case runtime.GOOS != "windows" && st.Mode()&0o111 == 0:
		c.Status = StatusFail
		c.Detail = "not executable: " + p
		c.Fix = "chmod +x " + p
	default:
		c.Status = StatusOK
		c.Detail = p
	}
	return c
}
