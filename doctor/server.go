package doctor

import (
	"os"
	"runtime"
	"strconv"

	"llamaherd/control"
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
	if w, ok := control.ParseWSLBin(p); ok {
		return checkWSLServerBin(c, w)
	}
	st, err := os.Stat(p)
	switch {
	case err != nil:
		c.Status = StatusFail
		c.Detail = "not found at " + p
		c.Fix = "install llama.cpp, then set the path:\n" +
			"  LLAMAHERD_SERVER_BIN=/path/to/llama-server llamaherd doctor\n" +
			wslHint() +
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

// wslHint tells Windows users that a llama-server inside WSL can be used.
func wslHint() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	return "  (llama-server inside WSL? use LLAMAHERD_SERVER_BIN=wsl:<distro>:/path/to/llama-server)\n"
}

// checkWSLServerBin verifies a "wsl:" server binary is executable in the distro.
func checkWSLServerBin(c Check, w control.WSLBin) Check {
	if runtime.GOOS != "windows" {
		c.Status = StatusFail
		c.Detail = w.String() + " is only valid when LlamaHerd runs on Windows"
		c.Fix = "inside WSL/Linux use the plain Linux path of llama-server"
		return c
	}
	name, args := w.Run("test", "-x", w.Path)
	if _, err := execOut(name, args...); err != nil {
		c.Status = StatusFail
		c.Detail = "not executable inside WSL: " + w.String()
		c.Fix = "check the distro name (wsl -l -v) and path, e.g.\n" +
			"  wsl -d <distro> -- ls -l " + w.Path
		return c
	}
	c.Status = StatusOK
	c.Detail = w.String() + " (runs through wsl.exe)"
	return c
}

// checkWSLModels warns when a WSL-hosted server would read models over the
// slow /mnt/<drive> bridge.
func checkWSLModels() Check {
	c := Check{ID: "wsl-models", Title: "WSL model access"}
	if _, ok := control.ParseWSLBin(settings.ServerBin()); !ok {
		c.Status, c.Detail = StatusSkip, "llama-server is not in WSL"
		return c
	}
	d := settings.ModelsDir()
	if control.IsWindowsDrivePath(d) {
		c.Status = StatusWarn
		c.Detail = "models are read through /mnt/" + "<drive> (slow first load): " + d
		c.Fix = "keep models inside WSL and point modelsDir at them via\n" +
			"  \\\\wsl.localhost\\<distro>\\home\\<you>\\models  (fast native reads)"
		return c
	}
	c.Status = StatusOK
	c.Detail = "models are read natively inside WSL: " + d
	return c
}
