package control

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProbeBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a /bin/sh script")
	}
	dir := t.TempDir()
	good := filepath.Join(dir, "llama-server")
	if err := os.WriteFile(good, []byte("#!/bin/sh\necho 'version: 1234 (abc)'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	v, err := ProbeBinary(good)
	if err != nil || !strings.Contains(v, "1234") {
		t.Fatalf("ProbeBinary = %q, %v", v, err)
	}

	bad := filepath.Join(dir, "broken")
	_ = os.WriteFile(bad, []byte("#!/bin/sh\necho boom >&2\nexit 7\n"), 0o755)
	if _, err := ProbeBinary(bad); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("broken probe = %v", err)
	}
	if _, err := ProbeBinary(filepath.Join(dir, "missing")); err == nil {
		t.Error("missing binary should fail")
	}
}
