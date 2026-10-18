package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"llamaherd/internal/ggufgen"
	"llamaherd/schema"
	"llamaherd/settings"
)

// isolateSettings points the models dir at a temp dir with no user config.
func isolateSettings(t *testing.T) string {
	t.Helper()
	md := t.TempDir()
	t.Cleanup(settings.Reload)
	t.Setenv(settings.EnvConfig, filepath.Join(t.TempDir(), "config.json"))
	t.Setenv(settings.EnvModelsDir, md)
	settings.Reload()
	return md
}

func samplePreset(model string) schema.Preset {
	return schema.Preset{
		Name: "default",
		Basic: schema.Basic{
			Model: model, Alias: "x", Host: "127.0.0.1", Port: 8090,
			Context: 4096, GPULayers: "all", KCache: "q8_0", VCache: "q8_0",
		},
		Advanced: schema.Advanced{SpecType: "draft-mtp", DraftNMax: 3},
	}
}

func TestSaveLoadAliasRoundTripAndPortable(t *testing.T) {
	md := isolateSettings(t)
	dir := t.TempDir()
	abs := filepath.Join(md, "sub", "m.gguf")
	if err := SaveAlias(dir, "x", []schema.Preset{samplePreset(abs)}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "x.json"))
	if strings.Contains(string(raw), md) {
		t.Errorf("saved file leaks the absolute models dir:\n%s", raw)
	}
	f, err := LoadAlias(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if f.Alias != "x" || len(f.Presets) != 1 {
		t.Fatalf("loaded %+v", f)
	}
	if got := f.Presets[0].Basic.Model; got != abs {
		t.Errorf("model path after load = %q, want %q", got, abs)
	}
	if f.Presets[0].Advanced.SpecType != "draft-mtp" || f.Presets[0].Advanced.DraftNMax != 3 {
		t.Errorf("advanced fields lost: %+v", f.Presets[0].Advanced)
	}
}

func TestLoadAliasErrors(t *testing.T) {
	isolateSettings(t)
	dir := t.TempDir()
	if _, err := LoadAlias(dir, "missing"); err == nil {
		t.Error("missing file should error")
	}
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{not json"), 0o644)
	if _, err := LoadAlias(dir, "bad"); err == nil {
		t.Error("malformed JSON should error")
	}
	os.WriteFile(filepath.Join(dir, "empty.json"), []byte(`{"alias":"empty","presets":[]}`), 0o644)
	if _, err := LoadAlias(dir, "empty"); err == nil {
		t.Error("a file with no presets should error")
	}
	if err := SaveAlias(dir, "none", nil); err == nil {
		t.Error("saving an empty preset list must be refused")
	}
}

func TestMetaRoundTripAndMalformed(t *testing.T) {
	dir := t.TempDir()
	m := Meta{ID: "a", Path: "a.gguf", Category: []string{"Qwen", "3.8"}, MTP: true, Source: "hf"}
	if err := SaveMeta(dir, m); err != nil {
		t.Fatal(err)
	}
	got, ok := LoadMeta(dir, "a")
	if !ok || got.Source != "hf" || len(got.Category) != 2 || !got.MTP {
		t.Fatalf("round trip: %+v ok=%v", got, ok)
	}
	if _, ok := LoadMeta(dir, "nope"); ok {
		t.Error("absent meta must report ok=false")
	}
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{"), 0o644)
	if _, ok := LoadMeta(dir, "bad"); ok {
		t.Error("malformed meta must report ok=false")
	}
	// ids can never escape the library directory
	if p := metaPath(dir, "../../etc/passwd"); filepath.Dir(p) != dir {
		t.Errorf("metaPath escaped the library dir: %s", p)
	}
}

func TestApplyLibraryCreatesStubsAndDetectsMTP(t *testing.T) {
	md := isolateSettings(t)
	plain, _ := ggufgen.Write(filepath.Join(md, "plain.gguf"), ggufgen.Spec{Layers: 8, Heads: 8, Embedding: 64})
	mtp, _ := ggufgen.Write(filepath.Join(md, "mtp.gguf"), ggufgen.Spec{Layers: 8, Heads: 8, Embedding: 64, NextNPredictLayers: 1})
	lib := t.TempDir()

	models := ScanModels(md)
	if n := ApplyLibrary(lib, models); n != 2 {
		t.Fatalf("created %d stubs, want 2", n)
	}
	byPath := map[string]schema.Model{}
	for _, m := range models {
		byPath[m.Path] = m
	}
	if byPath[plain].MTP {
		t.Error("plain model must not be flagged MTP")
	}
	if !byPath[mtp].MTP {
		t.Error("embedded NextN head must flag MTP")
	}
	if n := ApplyLibrary(lib, ScanModels(md)); n != 0 {
		t.Errorf("second pass created %d stubs, want 0", n)
	}
}
