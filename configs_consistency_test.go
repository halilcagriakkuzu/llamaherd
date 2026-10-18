package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"llamaherd/control"
	"llamaherd/dummy"
	"llamaherd/settings"
	"llamaherd/store"
)

var (
	validSpecTypes = map[string]bool{"": true, "none": true, "draft-mtp": true, "draft-simple": true}
	validFlashAttn = map[string]bool{"": true, "on": true, "off": true, "auto": true}
)

// TestBuiltinPresetsConsistent is the GPU-free, model-free guard: it validates
// the shape and generated command line of every built-in preset, so a broken
// shipped preset fails CI even on a machine with no GPU (TestAllPresetsFitBudget
// needs a real GPU + model files and skips without them).
func TestBuiltinPresetsConsistent(t *testing.T) {
	if len(dummy.Presets) == 0 {
		t.Fatal("no built-in presets")
	}
	for alias, list := range dummy.Presets {
		if len(list) == 0 {
			t.Errorf("%s: no presets", alias)
		}
		for i, p := range list {
			where := alias + "[" + strconv.Itoa(i) + "]/" + p.Name
			if p.Name == "" {
				t.Errorf("%s: empty preset name", where)
			}
			if p.Basic.Alias != alias {
				t.Errorf("%s: alias %q does not match key %q", where, p.Basic.Alias, alias)
			}
			if p.Basic.Model == "" {
				t.Errorf("%s: empty model", where)
			}
			if p.Basic.Host == "" {
				t.Errorf("%s: empty host", where)
			}
			if p.Basic.Port < 1 || p.Basic.Port > 65535 {
				t.Errorf("%s: port %d out of range", where, p.Basic.Port)
			}
			if p.Basic.Context <= 0 {
				t.Errorf("%s: context must be > 0, got %d", where, p.Basic.Context)
			}
			if !validGpuLayers(p.Basic.GPULayers) {
				t.Errorf("%s: invalid -ngl %q", where, p.Basic.GPULayers)
			}
			if !validFlashAttn[p.Basic.FlashAttention] {
				t.Errorf("%s: invalid flashAttention %q", where, p.Basic.FlashAttention)
			}
			for label, ct := range map[string]string{"kCache": p.Basic.KCache, "vCache": p.Basic.VCache} {
				if ct == "" {
					continue
				}
				if _, ok := control.BytesPerElement(ct); !ok {
					t.Errorf("%s: unknown %s quant %q", where, label, ct)
				}
			}
			if !validSpecTypes[p.Advanced.SpecType] {
				t.Errorf("%s: invalid specType %q", where, p.Advanced.SpecType)
			}
			if p.Advanced.SpecType == "draft-simple" && p.Advanced.DraftModel == "" {
				t.Errorf("%s: specType draft-simple requires a draftModel", where)
			}
			assertArgs(t, where, control.BuildArgs(p.AsConfig()))
		}
	}
}

func validGpuLayers(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "all", "auto":
		return true
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	return err == nil && n >= 0
}

// assertArgs checks a generated command line for empty tokens and that the
// required value flags are present.
func assertArgs(t *testing.T, where string, args []string) {
	t.Helper()
	for i, a := range args {
		if strings.TrimSpace(a) == "" {
			t.Errorf("%s: empty arg at index %d (%v)", where, i, args)
		}
	}
	for _, flag := range []string{"--model", "--host", "-ngl"} {
		if !hasFlagValue(args, flag) {
			t.Errorf("%s: %s missing or without a value (%v)", where, flag, args)
		}
	}
}

func hasFlagValue(args []string, flag string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] != "" {
			return true
		}
	}
	return false
}

// TestPresetSaveLoadRoundTrip checks configs/<alias>.json persistence: the
// preset set survives a save/load cycle and model paths are stored relative to
// the models directory (portable), with no GPU or real model files needed.
func TestPresetSaveLoadRoundTrip(t *testing.T) {
	md := t.TempDir()
	// Restore settings after the test; register before t.Setenv so the env is
	// restored first and the cache is reloaded against the original config.
	t.Cleanup(settings.Reload)
	t.Setenv(settings.EnvConfig, filepath.Join(t.TempDir(), "config.json"))
	t.Setenv(settings.EnvModelsDir, md)
	settings.Reload()

	dir := t.TempDir()
	for alias, ps := range dummy.Presets {
		if err := store.SaveAlias(dir, alias, ps); err != nil {
			t.Fatalf("%s: save: %v", alias, err)
		}
		f, err := store.LoadAlias(dir, alias)
		if err != nil {
			t.Fatalf("%s: load: %v", alias, err)
		}
		want := store.ResolvePresetPaths(ps)
		if len(f.Presets) != len(want) {
			t.Fatalf("%s: got %d presets, want %d", alias, len(f.Presets), len(want))
		}
		for i := range want {
			if !reflect.DeepEqual(f.Presets[i], want[i]) {
				t.Errorf("%s[%d]: round-trip mismatch\n got %+v\nwant %+v", alias, i, f.Presets[i], want[i])
			}
		}
	}
}

// TestPresetAbsolutePathsAreMadePortable proves a preset saved with an absolute
// model path under the models dir is written relative to it (so configs never
// capture a machine-specific path).
func TestPresetAbsolutePathsAreMadePortable(t *testing.T) {
	md := t.TempDir()
	t.Cleanup(settings.Reload)
	t.Setenv(settings.EnvConfig, filepath.Join(t.TempDir(), "config.json"))
	t.Setenv(settings.EnvModelsDir, md)
	settings.Reload()

	abs := filepath.Join(md, "sub", "x.gguf")
	ps := []dummy.Preset{{
		Name: "p",
		Basic: dummy.Basic{
			Model: abs, Alias: "portable", Host: "127.0.0.1",
			Port: 8090, Context: 4096, GPULayers: "all",
		},
		Advanced: dummy.Advanced{MMProj: filepath.Join(md, "mmproj.gguf")},
	}}
	dir := t.TempDir()
	if err := store.SaveAlias(dir, "portable", ps); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "portable.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), md) {
		t.Errorf("saved config leaked the absolute models dir %q:\n%s", md, raw)
	}
	f, err := store.LoadAlias(dir, "portable")
	if err != nil {
		t.Fatal(err)
	}
	if f.Presets[0].Basic.Model != abs {
		t.Errorf("resolved model = %q, want %q", f.Presets[0].Basic.Model, abs)
	}
	if f.Presets[0].Advanced.MMProj != filepath.Join(md, "mmproj.gguf") {
		t.Errorf("resolved mmproj = %q", f.Presets[0].Advanced.MMProj)
	}
}
