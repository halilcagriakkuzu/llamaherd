package control

import (
	"reflect"
	"testing"
)

func TestParseWSLBin(t *testing.T) {
	cases := map[string]WSLBin{
		"wsl:/opt/ls":        {Path: "/opt/ls"},
		"wsl:Ubuntu:/opt/ls": {Distro: "Ubuntu", Path: "/opt/ls"},
		"WSL:Debian-12:/x/y": {Distro: "Debian-12", Path: "/x/y"},
	}
	for in, want := range cases {
		got, ok := ParseWSLBin(in)
		if !ok || got != want {
			t.Errorf("%q = %+v,%v want %+v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "/usr/bin/llama-server", `C:\x\llama-server.exe`, "wsl:Ubuntu", "wsl:Ubuntu:rel/path"} {
		if _, ok := ParseWSLBin(bad); ok {
			t.Errorf("%q should not parse", bad)
		}
	}
}

func TestToWSLPath(t *testing.T) {
	cases := map[string]string{
		`C:\Users\me\LlamaHerd\models\a.gguf`:   "/mnt/c/Users/me/LlamaHerd/models/a.gguf",
		`d:/m/b.gguf`:                           "/mnt/d/m/b.gguf",
		`\\wsl.localhost\Ubuntu\home\me\m.gguf`: "/home/me/m.gguf",
		`\\wsl$\Ubuntu\home\me`:                 "/home/me",
		"/already/linux":                        "/already/linux",
		"8090":                                  "8090",
		"--model":                               "--model",
	}
	for in, want := range cases {
		if got := ToWSLPath(in); got != want {
			t.Errorf("ToWSLPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWSLCommand(t *testing.T) {
	w := WSLBin{Distro: "Ubuntu", Path: "/opt/llama-server"}
	name, args := w.Command([]string{"--model", `C:\m\a.gguf`, "--port", "8090"})
	want := []string{"-d", "Ubuntu", "--exec", "/opt/llama-server", "--model", "/mnt/c/m/a.gguf", "--port", "8090"}
	if name != "wsl.exe" || !reflect.DeepEqual(args, want) {
		t.Errorf("got %s %v", name, args)
	}
	_, stop := w.StopCommand(8090)
	if stop[len(stop)-1] != `/opt/llama-server .*--port 8090( |$)` {
		t.Errorf("stop pattern: %v", stop)
	}
}
