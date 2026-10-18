package doctor

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reBuild  = regexp.MustCompile(`(?i)build\s+(\d+)`)
	reCommit = regexp.MustCompile(`(?i)commit\s+([0-9a-f]{6,})`)
	reSemver = regexp.MustCompile(`^v?(\d+\.\d+\.\d+[-+.\w]*)`)
)

// parseLlamaVersion extracts version/build/commit from the output of
// `llama-server --version`, e.g.
// "version: 0.7.0-dev (build 11708, commit 83f938941)".
func parseLlamaVersion(out string) (version string, build int, commit string) {
	for _, line := range strings.Split(out, "\n") {
		i := strings.Index(strings.ToLower(line), "version:")
		if i < 0 {
			continue
		}
		rest := strings.TrimSpace(line[i+len("version:"):])
		if m := reSemver.FindStringSubmatch(rest); m != nil {
			version = m[1]
		}
		if m := reBuild.FindStringSubmatch(rest); m != nil {
			fmt.Sscanf(m[1], "%d", &build)
		}
		if m := reCommit.FindStringSubmatch(rest); m != nil {
			commit = m[1]
		}
	}
	return
}
