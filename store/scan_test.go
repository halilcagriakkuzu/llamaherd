package store

import (
	"os"
	"path/filepath"
	"testing"

	"llamaherd/schema"
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

func TestUniqueIDsDisambiguateDuplicateBaseNames(t *testing.T) {
	models := []schema.Model{
		{Path: "/models/a/model.gguf"},
		{Path: "/models/b/model.gguf"},
		{Path: "/models/c/Other.gguf"},
	}
	ids := UniqueIDs(models)
	if got := ids[0]; got != "model" {
		t.Errorf("first duplicate = %q, want model", got)
	}
	if ids[1] == ids[0] || ids[1] == "model" {
		t.Errorf("second duplicate not disambiguated: %q", ids[1])
	}
	if ids[2] != "other" {
		t.Errorf("non-duplicate = %q, want other", ids[2])
	}
	if again := UniqueIDs(models); again[1] != ids[1] {
		t.Errorf("ids are not stable: %q vs %q", again[1], ids[1])
	}
}

func TestQuantFromName(t *testing.T) {
	cases := map[string]string{
		"Qwen3.8-27B-UD-IQ4_XS.gguf": "IQ4_XS",
		"model-Q4_K_M.gguf":          "Q4_K_M",
		"tiny-Q8_0.gguf":             "Q8_0",
		"vision-F16.gguf":            "F16",
		"no-quant-here.gguf":         "",
	}
	for name, want := range cases {
		if got := QuantFromName(name); got != want {
			t.Errorf("QuantFromName(%q) = %q, want %q", name, got, want)
		}
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

func TestUniqueIDsCaseOnlyNames(t *testing.T) {
	models := []schema.Model{
		{Path: filepath.Join("A", "Model.gguf")},
		{Path: filepath.Join("B", "model.gguf")},
	}
	ids := UniqueIDs(models)
	if ids[0] == ids[1] {
		t.Fatalf("case-only names share an id: %v", ids)
	}
	if ids[0] != "model" || ids[1] == "model" {
		t.Errorf("unexpected ids: %v", ids)
	}
}
