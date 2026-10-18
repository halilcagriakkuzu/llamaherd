package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"llamaherd/control"
	"llamaherd/settings"
	"llamaherd/store"
)

// libMu serializes the library edits below (rename / move / category ops).
var libMu sync.Mutex

// modelIndex resolves a model reference (file path or alias) to its index in
// a.models; mmproj sidecars are never addressable.
func (a *App) modelIndex(ref string) (int, error) {
	for i, m := range a.models {
		if m.IsMMProj {
			continue
		}
		if m.Path == ref || m.Alias == ref {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no model matching %q", ref)
}

// serverUsesAlias reports whether the running (or starting) server was launched
// from the given alias, in which case renaming it would orphan the process.
func (a *App) serverUsesAlias(alias string) bool {
	st := a.mgr.Info().Status
	if st == control.StatusStopped || st == control.StatusCrashed {
		return false
	}
	return a.runCfg != nil && a.runCfg.Basic.Alias == alias
}

// RenameModel sets a model's display name and alias. An empty name restores the
// file-derived display name; an empty alias restores the file-derived alias.
// Changing the alias moves configs/<old>.json to configs/<new>.json and rewrites
// the --alias inside it. The .gguf file is never touched.
func (a *App) RenameModel(ref, name, alias string) error {
	libMu.Lock()
	defer libMu.Unlock()

	i, err := a.modelIndex(ref)
	if err != nil {
		return err
	}
	m := a.models[i]
	oldAlias := m.Alias

	name = strings.TrimSpace(name)
	if len(name) > 120 {
		return fmt.Errorf("display name is too long (max 120 characters)")
	}
	derived := store.DerivedAlias(m.Path)
	newAlias := derived
	if strings.TrimSpace(alias) != "" {
		if newAlias, err = store.NormalizeAlias(alias); err != nil {
			return err
		}
	}

	if newAlias != oldAlias {
		if a.serverUsesAlias(oldAlias) {
			return fmt.Errorf("stop the running server before renaming %q", oldAlias)
		}
		for j, o := range a.models {
			if j != i && !o.IsMMProj && o.Alias == newAlias {
				return fmt.Errorf("alias %q is already used by %q", newAlias, o.Display)
			}
		}
		newFile := filepath.Join(settings.ConfigsDir(), newAlias+".json")
		if _, err := os.Stat(newFile); err == nil {
			return fmt.Errorf("configs/%s.json already exists; pick another alias", newAlias)
		}
		// Move the presets under the new alias.
		ps := append(a.presets[oldAlias][:0:0], a.presets[oldAlias]...)
		for k := range ps {
			ps[k].Basic.Alias = newAlias
		}
		if len(ps) > 0 {
			if err := store.SaveAlias(settings.ConfigsDir(), newAlias, ps); err != nil {
				return err
			}
			_ = os.Remove(filepath.Join(settings.ConfigsDir(), oldAlias+".json"))
		}
		delete(a.presets, oldAlias)
		a.presets[newAlias] = ps
		if a.selModel == i && a.cfg.Basic.Alias == oldAlias {
			a.cfg.Basic.Alias = newAlias
		}
		a.models[i].Alias = newAlias
	}

	err = store.UpdateMeta(settings.LibraryDir(), m.Path, func(meta *store.Meta) {
		meta.Name = name
		meta.Alias = ""
		if newAlias != derived {
			meta.Alias = newAlias
		}
	})
	if err != nil {
		return err
	}
	if name != "" {
		a.models[i].Display = name
	} else {
		a.models[i].Display = store.DisplayName(m.Path)
	}
	a.Log(fmt.Sprintf("library: renamed %q → name %q, alias %q", oldAlias, a.models[i].Display, newAlias))
	a.emitState()
	return nil
}

// SetModelCategory places a model in the given category path; an empty path
// removes it from its category (it returns to "Uncategorized").
func (a *App) SetModelCategory(ref string, category []string) error {
	libMu.Lock()
	defer libMu.Unlock()

	i, err := a.modelIndex(ref)
	if err != nil {
		return err
	}
	clean, err := store.CleanCategoryPath(category)
	if err != nil {
		return err
	}
	if err := store.UpdateMeta(settings.LibraryDir(), a.models[i].Path, func(meta *store.Meta) {
		meta.Category = clean
	}); err != nil {
		return err
	}
	a.models[i].Category = clean
	a.Log(fmt.Sprintf("library: %s → %s", a.models[i].Display, categoryLabel(clean)))
	a.emitState()
	return nil
}

func categoryLabel(c []string) string {
	if len(c) == 0 {
		return store.Uncategorized
	}
	return strings.Join(c, " › ")
}

// categoryExists reports whether path names an existing category folder
// (through a model placed in or below it, or a persisted empty folder).
func (a *App) categoryExists(path []string) bool {
	for _, m := range a.models {
		if store.PathHasPrefix(m.Category, path) {
			return true
		}
	}
	for _, c := range a.cats {
		if store.PathHasPrefix(c, path) {
			return true
		}
	}
	return false
}

// CreateCategory adds an (initially empty) category folder.
func (a *App) CreateCategory(path []string) error {
	libMu.Lock()
	defer libMu.Unlock()

	clean, err := store.CleanCategoryPath(path)
	if err != nil {
		return err
	}
	if len(clean) == 0 {
		return fmt.Errorf("category path is empty")
	}
	if a.categoryExists(clean) {
		return fmt.Errorf("category %q already exists", categoryLabel(clean))
	}
	a.cats = append(a.cats, clean)
	if err := store.SaveCategories(settings.LibraryDir(), a.cats); err != nil {
		return err
	}
	a.Log("library: created category " + categoryLabel(clean))
	a.emitState()
	return nil
}

// RenameCategory renames the last segment of path, moving every model and empty
// folder below it. A sibling with the same name is refused (no silent merge).
func (a *App) RenameCategory(path []string, newName string) error {
	libMu.Lock()
	defer libMu.Unlock()

	if len(path) == 0 {
		return fmt.Errorf("category path is empty")
	}
	name, err := store.ValidCategoryName(newName)
	if err != nil {
		return err
	}
	if !a.categoryExists(path) {
		return fmt.Errorf("category %q does not exist", categoryLabel(path))
	}
	target := append(append([]string(nil), path[:len(path)-1]...), name)
	if !store.PathEqual(target, path) && a.categoryExists(target) {
		return fmt.Errorf("a category named %q already exists there", name)
	}
	for i := range a.models {
		m := a.models[i]
		if m.IsMMProj || !store.PathHasPrefix(m.Category, path) {
			continue
		}
		nc := store.RenamePathPrefix(m.Category, path, name)
		if err := store.UpdateMeta(settings.LibraryDir(), m.Path, func(meta *store.Meta) { meta.Category = nc }); err != nil {
			return err
		}
		a.models[i].Category = nc
	}
	for k, c := range a.cats {
		a.cats[k] = store.RenamePathPrefix(c, path, name)
	}
	if err := store.SaveCategories(settings.LibraryDir(), a.cats); err != nil {
		return err
	}
	a.Log(fmt.Sprintf("library: renamed category %s → %s", categoryLabel(path), name))
	a.emitState()
	return nil
}

// DeleteCategory removes a category folder. Models inside (at any depth) are not
// deleted: they move up to the parent category (or Uncategorized).
func (a *App) DeleteCategory(path []string) error {
	libMu.Lock()
	defer libMu.Unlock()

	if len(path) == 0 {
		return fmt.Errorf("category path is empty")
	}
	if !a.categoryExists(path) {
		return fmt.Errorf("category %q does not exist", categoryLabel(path))
	}
	parent := append([]string(nil), path[:len(path)-1]...)
	moved := 0
	for i := range a.models {
		m := a.models[i]
		if m.IsMMProj || !store.PathHasPrefix(m.Category, path) {
			continue
		}
		if err := store.UpdateMeta(settings.LibraryDir(), m.Path, func(meta *store.Meta) { meta.Category = parent }); err != nil {
			return err
		}
		a.models[i].Category = parent
		moved++
	}
	kept := a.cats[:0:0]
	for _, c := range a.cats {
		if !store.PathHasPrefix(c, path) {
			kept = append(kept, c)
		}
	}
	a.cats = kept
	if err := store.SaveCategories(settings.LibraryDir(), a.cats); err != nil {
		return err
	}
	a.Log(fmt.Sprintf("library: deleted category %s (%d model(s) moved to %s)",
		categoryLabel(path), moved, categoryLabel(parent)))
	a.emitState()
	return nil
}
