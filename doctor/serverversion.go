package doctor

import (
	"fmt"
	"os"

	"llamaherd/control"
	"llamaherd/settings"
)

func checkServerVersion() Check {
	c := Check{ID: "server-version", Title: "llama-server version"}
	p := settings.ServerBin()
	name, args := p, []string{"--version"}
	if w, ok := control.ParseWSLBin(p); ok {
		name, args = w.Run(w.Path, "--version")
	} else if _, err := os.Stat(p); err != nil {
		c.Status, c.Detail = StatusSkip, "binary not found"
		return c
	}
	out, err := execOut(name, args...)
	if err != nil && out == "" {
		c.Status, c.Detail = StatusWarn, "could not run --version"
		c.Fix = fmt.Sprintf("run it directly to see the error: %s --version", p)
		return c
	}
	ver, build, commit := parseLlamaVersion(out)
	desc := ver
	if desc == "" {
		desc = "unknown version"
	}
	if build > 0 {
		desc += fmt.Sprintf(" (build %d", build)
		if commit != "" {
			desc += ", commit " + commit
		}
		desc += ")"
	}
	c.Detail = desc
	if build > 0 && build < minBuild() {
		c.Status = StatusWarn
		if control.IsShallowRepo(settings.LlamaDir()) {
			c.Detail += " — shallow git clone, so the build number is not comparable"
			c.Fix = "the checkout at " + settings.LlamaDir() + " is shallow; run\n" +
				"  git -C " + settings.LlamaDir() + " fetch --unshallow\n" +
				"then rebuild (llama.cpp tab → Update, or scripts/update-llamacpp.sh)"
			return c
		}
		c.Detail += fmt.Sprintf(" — older than the recommended build %d", minBuild())
		c.Fix = "update llama.cpp with scripts/update-llamacpp.sh (or set LLAMAHERD_MIN_LLAMA_BUILD)"
		return c
	}
	c.Status = StatusOK
	return c
}
