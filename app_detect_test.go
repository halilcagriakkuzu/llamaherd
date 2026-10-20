package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"llamaherd/settings"
)

func TestDetectServerBin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a /bin/sh script")
	}
	a, root := libApp(t)
	bin := filepath.Join(root, "llama-server")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho 'version: 9999 (detect-test)'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(settings.EnvServerBin, bin)
	settings.Reload()

	d, err := a.DetectServerBin()
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != bin || d.Version == "" {
		t.Errorf("detected = %+v, want %s", d, bin)
	}
}
