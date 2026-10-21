package main

import (
	"testing"

	"llamaherd/settings"
	"llamaherd/store"
)

func TestRestoreLastRunSelectsModelAndPreset(t *testing.T) {
	a, root := libApp(t, "alpha", "beta")
	if len(a.models) < 2 {
		t.Fatalf("models = %+v", a.models)
	}
	target := a.models[1].Alias
	// Give the target a second preset so the test proves the preset name is used.
	list := a.presets[target]
	p := list[0]
	p.Name = "fast"
	a.presets[target] = append(list, p)
	if err := store.SaveAlias(settings.ConfigsDir(), target, a.presets[target]); err != nil {
		t.Fatal(err)
	}
	if err := settings.SaveLastRun(settings.LastRun{Alias: target, Preset: "fast"}); err != nil {
		t.Fatal(err)
	}
	a.selModel = 0
	if !a.restoreLastRun() {
		t.Fatal("restoreLastRun did not find the remembered model")
	}
	if a.alias() != target || a.selPreset != 1 {
		t.Fatalf("selection = %s/%d, want %s/1", a.alias(), a.selPreset, target)
	}
	if a.cfg.Basic.Alias != target {
		t.Fatalf("config alias = %s, want %s", a.cfg.Basic.Alias, target)
	}
	if root == "" {
		t.Fatal("missing temp root")
	}
}

func TestRestoreLastRunIgnoresDeletedModel(t *testing.T) {
	a, _ := libApp(t, "alpha")
	before := a.selModel
	if err := settings.SaveLastRun(settings.LastRun{Alias: "gone", Preset: "default"}); err != nil {
		t.Fatal(err)
	}
	if a.restoreLastRun() {
		t.Fatal("a deleted model must not be restored")
	}
	if a.selModel != before {
		t.Fatalf("selection changed from %d to %d", before, a.selModel)
	}
}
