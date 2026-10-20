package control

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// WSLPrefix marks a serverBin that lives inside WSL while LlamaHerd itself runs
// on native Windows:
//
//	wsl:/home/me/llama.cpp/build/bin/llama-server          (default distro)
//	wsl:Ubuntu:/home/me/llama.cpp/build/bin/llama-server   (named distro)
//
// The server is launched through `wsl.exe --exec`, Windows paths in its
// arguments are translated (C:\x -> /mnt/c/x, \\wsl.localhost\D\p -> /p), and
// it is reached over WSL2's localhost forwarding like any local server.
const WSLPrefix = "wsl:"

// WSLBin is a parsed WSL-hosted llama-server location.
type WSLBin struct {
	Distro string // empty = default distro
	Path   string // absolute Linux path
}

// ParseWSLBin parses a "wsl:" serverBin value.
func ParseWSLBin(s string) (WSLBin, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(strings.ToLower(s), WSLPrefix) {
		return WSLBin{}, false
	}
	rest := s[len(WSLPrefix):]
	if strings.HasPrefix(rest, "/") {
		return WSLBin{Path: rest}, true
	}
	i := strings.Index(rest, ":")
	if i <= 0 || !strings.HasPrefix(rest[i+1:], "/") {
		return WSLBin{}, false
	}
	return WSLBin{Distro: rest[:i], Path: rest[i+1:]}, true
}

func (w WSLBin) String() string {
	if w.Distro == "" {
		return WSLPrefix + w.Path
	}
	return WSLPrefix + w.Distro + ":" + w.Path
}

// wslArgs returns the `wsl.exe` argument prefix selecting the distro.
func (w WSLBin) wslArgs() []string {
	if w.Distro == "" {
		return nil
	}
	return []string{"-d", w.Distro}
}

// Command builds the wsl.exe invocation that runs the server with args
// (Windows paths in args are translated).
func (w WSLBin) Command(args []string) (string, []string) {
	out := append(w.wslArgs(), "--exec", w.Path)
	for _, a := range args {
		out = append(out, ToWSLPath(a))
	}
	return "wsl.exe", out
}

// Run builds a wsl.exe invocation of an arbitrary Linux command in the distro.
func (w WSLBin) Run(cmd ...string) (string, []string) {
	return "wsl.exe", append(append(w.wslArgs(), "--exec"), cmd...)
}

// StopCommand terminates the server serving port inside WSL (killing wsl.exe
// alone does not reliably stop the Linux process).
func (w WSLBin) StopCommand(port int) (string, []string) {
	pat := fmt.Sprintf("%s .*--port %d( |$)", regexp.QuoteMeta(w.Path), port)
	return w.Run("pkill", "-TERM", "-f", "--", pat)
}

var (
	driveRe = regexp.MustCompile(`^([A-Za-z]):[\\/](.*)$`)
	uncRe   = regexp.MustCompile(`(?i)^\\\\(?:wsl\.localhost|wsl\$)\\[^\\]+(\\.*)?$`)
)

// ToWSLPath converts a Windows path to its path inside WSL; anything that is
// not a Windows drive/WSL-UNC path is returned unchanged.
func ToWSLPath(p string) string {
	if m := driveRe.FindStringSubmatch(p); m != nil {
		return "/mnt/" + strings.ToLower(m[1]) + "/" + strings.ReplaceAll(m[2], `\`, "/")
	}
	if m := uncRe.FindStringSubmatch(p); m != nil {
		if m[1] == "" {
			return "/"
		}
		return strings.ReplaceAll(m[1], `\`, "/")
	}
	return p
}

// IsWindowsDrivePath reports whether p lives on a Windows drive (a path WSL
// reaches through the slow /mnt/<drive> bridge).
func IsWindowsDrivePath(p string) bool { return driveRe.MatchString(p) }

// BinaryExists reports whether the llama-server binary can be run: a plain
// file check, or `test -x` inside the distro for a "wsl:" binary.
func BinaryExists(bin string) bool {
	if w, ok := ParseWSLBin(bin); ok {
		name, args := w.Run("test", "-x", w.Path)
		return exec.Command(name, args...).Run() == nil
	}
	_, err := os.Stat(bin)
	return err == nil
}

// DiscoverWSLBinaries looks for llama-server inside each installed WSL distro
// (Windows only). It is best-effort and returns "wsl:<distro>:<path>" values.
func DiscoverWSLBinaries() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	out, err := exec.Command("wsl.exe", "-l", "-q").Output()
	if err != nil {
		return nil
	}
	var found []string
	for _, line := range strings.Split(string(out), "\n") {
		distro := strings.TrimSpace(strings.TrimRight(line, "\r"))
		if distro == "" {
			continue
		}
		w := WSLBin{Distro: distro}
		name, args := w.Run("sh", "-lc", "command -v llama-server 2>/dev/null || true")
		probe, err := exec.Command(name, args...).Output()
		if err != nil {
			continue
		}
		p := strings.TrimSpace(string(probe))
		if i := strings.LastIndexByte(p, '\n'); i >= 0 {
			p = strings.TrimSpace(p[i+1:])
		}
		if strings.HasPrefix(p, "/") {
			found = append(found, (WSLBin{Distro: distro, Path: p}).String())
		}
	}
	return found
}
