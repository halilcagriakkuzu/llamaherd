package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"llamaherd/internal/ggufgen"
	"llamaherd/settings"
	"llamaherd/store"
)

// libApp builds an App over a temp models dir holding the named (synthetic)
// models, with every settings path isolated from the real machine.
func libApp(t *testing.T, names ...string) (*App, string) {
	t.Helper()
	root := t.TempDir()
	models := filepath.Join(root, "models")
	for _, n := range names {
		if _, err := ggufgen.Write(filepath.Join(models, n+".gguf"),
			ggufgen.Spec{Layers: 8, Heads: 8, KVHeads: 8, Embedding: 512}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(settings.Reload)
	t.Setenv(settings.EnvConfig, filepath.Join(root, "config.json"))
	t.Setenv(settings.EnvModelsDir, models)
	t.Setenv(settings.EnvDataDir, filepath.Join(root, "data"))
	t.Setenv(settings.EnvConfigsDir, filepath.Join(root, "configs"))
	t.Setenv(settings.EnvLibraryDir, filepath.Join(root, "library"))
	t.Setenv(settings.EnvServerBin, filepath.Join(root, "nonexistent-llama-server"))
	settings.Reload()
	return reload(), root
}

func reload() *App {
	a := NewApp()
	a.initBackend()
	a.initModels()
	a.initPresets()
	return a
}

func modelByAlias(a *App, alias string) (int, bool) {
	for i, m := range a.models {
		if m.Alias == alias {
			return i, true
		}
	}
	return -1, false
}

func TestRenameModelAliasMovesConfigsAndPersists(t *testing.T) {
	a, root := libApp(t, "alpha", "beta")
	oldFile := filepath.Join(root, "configs", "alpha.json")
	if _, err := os.Stat(oldFile); err != nil {
		t.Fatalf("expected seeded configs/alpha.json: %v", err)
	}

	if err := a.RenameModel("alpha", "My Alpha", "Fast-Alpha"); err != nil {
		t.Fatal(err)
	}
	i, ok := modelByAlias(a, "fast-alpha") // alias is lower-cased
	if !ok || a.models[i].Display != "My Alpha" {
		t.Fatalf("model not renamed: %+v", a.models)
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old configs file should be gone")
	}
	f, err := store.LoadAlias(filepath.Join(root, "configs"), "fast-alpha")
	if err != nil || f.Presets[0].Basic.Alias != "fast-alpha" {
		t.Fatalf("new configs file wrong: %+v err=%v", f, err)
	}

	// A fresh start must see the same names (they live in library/<id>.json).
	b := reload()
	j, ok := modelByAlias(b, "fast-alpha")
	if !ok || b.models[j].Display != "My Alpha" {
		t.Fatalf("rename did not persist: %+v", b.models)
	}
	if _, ok := modelByAlias(b, "alpha"); ok {
		t.Error("old alias must not come back after restart")
	}

	// Empty name/alias restores the file-derived values.
	if err := b.RenameModel("fast-alpha", "", ""); err != nil {
		t.Fatal(err)
	}
	if k, ok := modelByAlias(b, "alpha"); !ok || b.models[k].Display != "alpha" {
		t.Errorf("reset failed: %+v", b.models)
	}
	if _, err := os.Stat(oldFile); err != nil {
		t.Errorf("configs/alpha.json should be back: %v", err)
	}
}

func TestRenameModelRejectsBadInput(t *testing.T) {
	a, _ := libApp(t, "alpha", "beta")
	cases := []struct{ ref, name, alias, want string }{
		{"alpha", "", "beta", "already used"},
		{"alpha", "", "Bad Alias!", "invalid"},
		{"alpha", strings.Repeat("x", 200), "", "too long"},
		{"missing", "", "", "no model"},
	}
	for _, c := range cases {
		err := a.RenameModel(c.ref, c.name, c.alias)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("RenameModel(%q,%q,%q) = %v, want error containing %q", c.ref, c.name, c.alias, err, c.want)
		}
	}
	if _, ok := modelByAlias(a, "alpha"); !ok {
		t.Error("failed renames must leave the model untouched")
	}
}

func TestCategoriesLifecycle(t *testing.T) {
	a, _ := libApp(t, "alpha", "beta", "gamma")

	if err := a.SetModelCategory("alpha", []string{"Qwen", "3.8"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetModelCategory("beta", []string{"Qwen"}); err != nil {
		t.Fatal(err)
	}
	if err := a.CreateCategory([]string{"Empty", "Sub"}); err != nil {
		t.Fatal(err)
	}
	if err := a.CreateCategory([]string{"Qwen", "3.8"}); err == nil {
		t.Error("creating an existing category must fail")
	}
	if err := a.CreateCategory([]string{"Uncategorized"}); err == nil {
		t.Error("the reserved name must be refused")
	}

	// rename Qwen -> Alibaba moves models at every depth
	if err := a.RenameCategory([]string{"Qwen"}, "Alibaba"); err != nil {
		t.Fatal(err)
	}
	ai, _ := modelByAlias(a, "alpha")
	bi, _ := modelByAlias(a, "beta")
	if !reflect.DeepEqual(a.models[ai].Category, []string{"Alibaba", "3.8"}) ||
		!reflect.DeepEqual(a.models[bi].Category, []string{"Alibaba"}) {
		t.Fatalf("categories after rename: %v / %v", a.models[ai].Category, a.models[bi].Category)
	}
	if err := a.RenameCategory([]string{"Alibaba"}, "Empty"); err == nil {
		t.Error("renaming onto an existing sibling must fail")
	}

	// persisted: a restarted app sees the same placement and the empty folder
	b := reload()
	ai, _ = modelByAlias(b, "alpha")
	if !reflect.DeepEqual(b.models[ai].Category, []string{"Alibaba", "3.8"}) {
		t.Errorf("category not persisted: %v", b.models[ai].Category)
	}
	if !b.categoryExists([]string{"Empty", "Sub"}) {
		t.Error("empty category must persist")
	}

	// deleting a populated category moves its models up, never deletes them
	if err := b.DeleteCategory([]string{"Alibaba", "3.8"}); err != nil {
		t.Fatal(err)
	}
	ai, _ = modelByAlias(b, "alpha")
	if !reflect.DeepEqual(b.models[ai].Category, []string{"Alibaba"}) {
		t.Errorf("model should move to the parent: %v", b.models[ai].Category)
	}
	if err := b.DeleteCategory([]string{"Empty"}); err != nil {
		t.Fatal(err)
	}
	if b.categoryExists([]string{"Empty", "Sub"}) {
		t.Error("deleting a category must remove its empty children")
	}
	if err := b.DeleteCategory([]string{"Nope"}); err == nil {
		t.Error("deleting an unknown category must fail")
	}

	// removing a model from its category returns it to Uncategorized
	if err := b.SetModelCategory("alpha", nil); err != nil {
		t.Fatal(err)
	}
	ai, _ = modelByAlias(b, "alpha")
	if len(b.models[ai].Category) != 0 {
		t.Errorf("category not cleared: %v", b.models[ai].Category)
	}
	if err := b.SetModelCategory("alpha", []string{"bad/name"}); err == nil {
		t.Error("invalid category names must be refused")
	}
}
