package store

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanModels(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "a", "Model-A-Q4.gguf"))
	touch(t, filepath.Join(root, "a", "mmproj-F16.gguf"))
	touch(t, filepath.Join(root, "b", "deep", "Model-B.GGUF"))
	touch(t, filepath.Join(root, "b", "Model-B-FastMTP-32K.gguf")) // MTP sidecar: not a model
	touch(t, filepath.Join(root, "notes.txt"))

	models := ScanModels(root)
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2: %+v", len(models), models)
	}
	a := models[0]
	if filepath.Base(a.Path) != "Model-A-Q4.gguf" {
		t.Fatalf("models not sorted by path: %+v", models)
	}
	if filepath.Base(a.MMProj) != "mmproj-F16.gguf" {
		t.Errorf("mmproj not attached to sibling model: %q", a.MMProj)
	}
	if models[1].MMProj != "" {
		t.Errorf("model without sibling mmproj got %q", models[1].MMProj)
	}
}

func TestScanModelsEmptyOrMissingDir(t *testing.T) {
	if got := ScanModels(t.TempDir()); len(got) != 0 {
		t.Errorf("empty dir: got %v", got)
	}
	if got := ScanModels(filepath.Join(t.TempDir(), "nope")); len(got) != 0 {
		t.Errorf("missing dir: got %v", got)
	}
}

func TestDerivedAlias(t *testing.T) {
	got := DerivedAlias("/a/b/Qwen3.8-27B-UD-IQ4_XS.gguf")
	want := "qwen3-8-27b-ud-iq4_xs"
	if got != want {
		t.Errorf("DerivedAlias = %q, want %q", got, want)
	}
}

func TestDisplayElidesLongNames(t *testing.T) {
	long := "Qwen3.8-27B-Uncensored-HauhauCS-Aggressive-Q3_K_P-extra-long-name"
	if d := display(long); len(d) >= len(long) || d[:16] != long[:16] {
		t.Errorf("display(%q) = %q", long, d)
	}
	if display("short") != "short" {
		t.Error("short names must be unchanged")
	}
}
