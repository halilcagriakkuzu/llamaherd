package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFramelessDefaultIsWSL(t *testing.T) {
	t.Cleanup(Reload)
	t.Setenv(EnvConfig, filepath.Join(t.TempDir(), "absent.json"))
	t.Setenv(EnvFrameless, "")
	Reload()
	if got, want := Frameless(), IsWSL(); got != want {
		t.Fatalf("Frameless default = %v, want %v (WSL=%v)", got, want, want)
	}
}

func TestFramelessEnvOverride(t *testing.T) {
	t.Cleanup(Reload)
	t.Setenv(EnvConfig, filepath.Join(t.TempDir(), "absent.json"))
	Reload()
	for _, v := range []string{"0", "false", "off", "no"} {
		t.Setenv(EnvFrameless, v)
		if Frameless() {
			t.Fatalf("LLAMAHERD_FRAMELESS=%q must disable the frameless window", v)
		}
	}
	for _, v := range []string{"1", "true", "on", "yes"} {
		t.Setenv(EnvFrameless, v)
		if !Frameless() {
			t.Fatalf("LLAMAHERD_FRAMELESS=%q must enable the frameless window", v)
		}
	}
}

func TestFramelessConfigKey(t *testing.T) {
	t.Cleanup(Reload)
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(`{"frameless": false}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvConfig, p)
	t.Setenv(EnvFrameless, "")
	Reload()
	if Frameless() {
		t.Fatal("config frameless=false must win over the WSL default")
	}

	// The env var still wins over the config file.
	t.Setenv(EnvFrameless, "1")
	if !Frameless() {
		t.Fatal("LLAMAHERD_FRAMELESS=1 must win over the config file")
	}
}
