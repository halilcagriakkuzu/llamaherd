package settings

import (
	"path/filepath"
	"testing"
)

func TestPathEqualModes(t *testing.T) {
	if !PathEqual("Models/A.gguf", "models/a.gguf", true) {
		t.Error("folded comparison should match case-only differences")
	}
	if PathEqual("Models/A.gguf", "models/a.gguf", false) {
		t.Error("exact comparison must stay case-sensitive")
	}
	if !PathEqual("models/./A.gguf", "models/A.gguf", false) {
		t.Error("comparison should clean . segments")
	}
}

func TestPathInsideModes(t *testing.T) {
	dir := filepath.Join("Models", "GGUF")
	if !PathInsideWith(dir, filepath.Join(dir, "sub", "a.gguf"), false) {
		t.Error("exact child should be inside")
	}
	if PathInsideWith(dir, filepath.Join("Models", "GGUF2", "a.gguf"), false) {
		t.Error("sibling prefix must not count as inside")
	}
	if !PathInsideWith(dir, filepath.Join("models", "gguf", "sub", "a.gguf"), true) {
		t.Error("case-folded child should be inside")
	}
	if PathInsideWith(dir, filepath.Join("Models", "Other", "a.gguf"), true) {
		t.Error("different folder must not be inside")
	}
}

func TestRelativizeModelPathCase(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(Reload)
	t.Setenv(EnvModelsDir, filepath.Join(root, "Models"))
	Reload()
	// Exact case always works.
	if got := RelativizeModelPath(filepath.Join(root, "Models", "A.gguf")); got != "A.gguf" {
		t.Errorf("exact = %q, want A.gguf", got)
	}
	// A case-only difference is resolved only on case-insensitive platforms;
	// on Linux the absolute path is kept unchanged.
	got := RelativizeModelPath(filepath.Join(root, "models", "A.gguf"))
	if CaseInsensitivePaths() {
		if got != "a.gguf" {
			t.Errorf("folded = %q, want a.gguf", got)
		}
	} else if !filepath.IsAbs(got) {
		t.Errorf("case-only path on Linux should stay absolute, got %q", got)
	}
}
