package main

import (
	"testing"

	"llamaherd/schema"
	"llamaherd/settings"
	"llamaherd/store"
)

func TestSavePresetAsRenameAndRemove(t *testing.T) {
	a, _ := libApp(t, "alpha")
	i, ok := modelByAlias(a, "alpha")
	if !ok {
		t.Fatal("model alpha missing")
	}
	a.SelectModel(i)

	a.cfg.Basic.Port = 9001
	if err := a.SavePresetAs("fast"); err != nil {
		t.Fatal(err)
	}
	if got := a.presetNames(); len(got) != 2 || got[0] != "default" || got[1] != "fast" {
		t.Fatalf("presets = %v", got)
	}
	if a.selPreset != 1 || a.dirty {
		t.Fatalf("new preset not selected/clean: sel=%d dirty=%v", a.selPreset, a.dirty)
	}

	if err := a.SavePresetAs("fast"); err == nil {
		t.Error("duplicate preset name should fail")
	}

	if err := a.RenamePreset("alpha", "fast", "faster"); err != nil {
		t.Fatal(err)
	}
	if got := a.presetNames(); len(got) != 2 || got[1] != "faster" {
		t.Fatalf("renamed presets = %v", got)
	}
	if a.selPreset != 1 {
		t.Fatalf("rename moved the selection: %d", a.selPreset)
	}
	if err := a.RenamePreset("alpha", "default", "faster"); err == nil {
		t.Error("rename onto an existing name should fail")
	}

	f, err := store.LoadAlias(settings.ConfigsDir(), "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Presets) != 2 || f.Presets[1].Name != "faster" || f.Presets[1].Basic.Port != 9001 {
		t.Fatalf("persisted presets = %+v", f.Presets)
	}

	// Removing a preset that is not applied must not move the selection.
	if err := a.RemovePreset("alpha", "default"); err != nil {
		t.Fatal(err)
	}
	if got := a.presetNames(); len(got) != 1 || got[0] != "faster" {
		t.Fatalf("presets after remove = %v", got)
	}
	if a.selPreset != 0 {
		t.Fatalf("selection after removing an earlier preset = %d, want 0", a.selPreset)
	}

	if err := a.RemovePreset("alpha", "faster"); err == nil {
		t.Error("removing the last preset should fail")
	}
}

func TestUpdatePresetMergesPartialConfig(t *testing.T) {
	a, _ := libApp(t, "alpha")
	base := a.presets["alpha"][0]
	if base.Basic.Model == "" || base.Basic.Context == 0 {
		t.Fatalf("default preset is incomplete: %+v", base)
	}

	patch := schema.Config{Basic: schema.Basic{Port: 9555}}
	if err := a.UpdatePreset("alpha", "default", patch); err != nil {
		t.Fatal(err)
	}
	got := a.presets["alpha"][0]
	if got.Basic.Port != 9555 {
		t.Errorf("port = %d, want 9555", got.Basic.Port)
	}
	if got.Basic.Model != base.Basic.Model || got.Basic.Context != base.Basic.Context ||
		got.Basic.GPULayers != base.Basic.GPULayers || got.Advanced.Fit != base.Advanced.Fit {
		t.Errorf("partial update dropped fields: got %+v, base %+v", got, base)
	}
	f, err := store.LoadAlias(settings.ConfigsDir(), "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if f.Presets[0].Basic.Port != 9555 || f.Presets[0].Basic.Model != base.Basic.Model {
		t.Errorf("merge not persisted: %+v", f.Presets[0])
	}
}
