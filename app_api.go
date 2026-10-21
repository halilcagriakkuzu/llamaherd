package main

// app_api.go — control surface for the HTTP API. These methods are
// the same operations the GUI bindings call, so the window reflects every
// CLI/API-driven change through the normal emitState path.

import (
	"encoding/json"
	"fmt"
	"strings"

	"llamaherd/control"
	"llamaherd/schema"
	"llamaherd/settings"
	"llamaherd/store"
)

// SnapshotJSON marshals the current snapshot (api.Control).
func (a *App) SnapshotJSON() ([]byte, error) {
	return json.Marshal(a.snapshot())
}

// Models returns the merged model list (api.Control).
func (a *App) Models() []schema.Model {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	out := append([]schema.Model(nil), a.models...)
	for i := range out {
		out[i].Category = append([]string(nil), out[i].Category...)
	}
	return out
}

// Presets returns the named presets of an alias (api.Control).
func (a *App) Presets(alias string) []schema.Preset {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return append([]schema.Preset(nil), a.presetList(alias)...)
}

// SelectModelByAlias selects the loadable model whose alias equals the given
// string or uniquely contains it as a substring (e.g. "27b-local" vs
// "-uncensored-27b-local"). Ambiguity is an error; mmproj entries never match.
func (a *App) SelectModelByAlias(alias string) error {
	if alias == "" {
		return nil // keep current selection
	}
	idx, err := a.resolveModelAlias(alias)
	if err != nil {
		return err
	}
	a.SelectModel(idx)
	return nil
}

// resolveModelAlias maps an alias/substring to a model index under the state
// lock. It is the shared lookup behind SelectModelByAlias and Open.
func (a *App) resolveModelAlias(alias string) (int, error) {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	exact, fuzzy := -1, -1
	count := 0
	for i, m := range a.models {
		if m.IsMMProj {
			continue
		}
		if strings.EqualFold(m.Alias, alias) {
			exact = i
			count++
			break
		}
		if strings.Contains(m.Alias, strings.ToLower(alias)) {
			fuzzy = i
			count++
		}
	}
	switch {
	case exact >= 0:
		return exact, nil
	case count == 1:
		return fuzzy, nil
	case count == 0:
		return -1, fmt.Errorf("no model alias matching %q (use `llamaherd list`)", alias)
	default:
		return -1, fmt.Errorf("alias %q is ambiguous (%d matches); use a longer string", alias, count)
	}
}

// Open selects (optionally) a model and preset, then starts the real server.
// alias and preset may be empty to act on the current selection.
func (a *App) Open(alias, preset string) error {
	if alias != "" {
		if err := a.SelectModelByAlias(alias); err != nil {
			return err
		}
	}
	if preset != "" {
		a.stateMu.Lock()
		curAlias := a.alias()
		if a.findPreset(curAlias, preset) == nil {
			a.stateMu.Unlock()
			return fmt.Errorf("no preset %q for alias %s (use `llamaherd list %s`)", preset, curAlias, curAlias)
		}
		a.applyPreset(a.presetIndex(preset))
		a.dirty = false
		a.stateMu.Unlock()
		a.Log(fmt.Sprintf("ctl: preset %s/%s", curAlias, preset))
		a.emitState()
	}
	a.stateMu.RLock()
	curAlias, curPreset := a.alias(), ""
	if names := a.presetNames(); len(names) > 0 {
		curPreset = names[a.selPreset]
	}
	a.stateMu.RUnlock()
	a.Log(fmt.Sprintf("ctl: open %s [%s]", curAlias, curPreset))
	a.StartServer()
	a.stateMu.RLock()
	warn := a.warn
	a.stateMu.RUnlock()
	if warn != "" {
		// StartServer refused (e.g. VRAM pre-flight): surface it to the caller.
		return fmt.Errorf("%s", warn)
	}
	return nil
}

// Stop stops the real server if we spawned it (api.Control).
func (a *App) Stop() error {
	info := a.mgr.Info()
	if info.Status == control.StatusStopped || info.Status == control.StatusCrashed {
		return fmt.Errorf("server is %s; nothing to stop", info.Status)
	}
	if err := a.mgr.Stop(); err != nil {
		a.Log("ctl: stop FAILED: " + err.Error())
		return err
	}
	a.Log("ctl: stop requested")
	a.emitState()
	return nil
}

// AddPreset appends a new named preset for the alias and persists it
// (configs/<alias>.json). Duplicate names are rejected.
func (a *App) AddPreset(alias, name string, cfg schema.Config) error {
	if name == "" {
		return fmt.Errorf("preset name required")
	}
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	list := a.presets[alias]
	for _, p := range list {
		if p.Name == name {
			return fmt.Errorf("preset %q already exists for %s (use config update)", name, alias)
		}
	}
	if cfg.Basic.Model == "" {
		// Seed from the alias's default preset so the new preset is complete.
		if first := a.findPreset(alias, firstName(list)); first != nil {
			c := first.AsConfig()
			c.Basic.Alias = alias
			cfg = mergeConfig(c, cfg)
		}
	}
	n := len(list)
	list = append(list, cfg.AsPreset(name))
	a.presets[alias] = list
	if err := store.SaveAlias(settings.ConfigsDir(), alias, list); err != nil {
		a.Log(fmt.Sprintf("ctl: save FAILED configs/%s.json: %v", alias, err))
		return err
	}
	a.Log(fmt.Sprintf("ctl: added preset %s/%s (configs/%s.json now has %d preset(s))",
		alias, name, alias, n+1))
	a.emitStateLocked()
	return nil
}

// UpdatePreset merges the supplied (possibly partial) config into the named
// preset and persists it; omitted fields keep their current values.
func (a *App) UpdatePreset(alias, name string, cfg schema.Config) error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	list := a.presets[alias]
	for i := range list {
		if list[i].Name != name {
			continue
		}
		// The API documents a partial update: keep every field the caller did
		// not supply, and always keep the preset name.
		merged := mergeConfig(list[i].AsConfig(), cfg)
		list[i] = merged.AsPreset(name)
		a.presets[alias] = list
		if err := store.SaveAlias(settings.ConfigsDir(), alias, list); err != nil {
			a.Log(fmt.Sprintf("ctl: save FAILED configs/%s.json: %v", alias, err))
			return err
		}
		// If it's the currently applied preset, refresh the forms.
		cur := a.presetNames()[a.selPreset]
		if cur == name && a.alias() == alias {
			a.applyPreset(a.selPreset)
		}
		a.Log(fmt.Sprintf("ctl: updated preset %s/%s (configs/%s.json)", alias, name, alias))
		a.emitStateLocked()
		return nil
	}
	return fmt.Errorf("no preset %q for alias %s", name, alias)
}

// RenamePreset changes a preset's name and persists it. Its contents and the
// current selection are unchanged.
func (a *App) RenamePreset(alias, oldName, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("preset name required")
	}
	if newName == oldName {
		return nil
	}
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	list := a.presets[alias]
	idx := -1
	for i := range list {
		if list[i].Name == oldName {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("no preset %q for alias %s", oldName, alias)
	}
	if a.findPreset(alias, newName) != nil {
		return fmt.Errorf("preset %q already exists for %s", newName, alias)
	}
	next := append([]schema.Preset(nil), list...)
	next[idx].Name = newName
	if err := store.SaveAlias(settings.ConfigsDir(), alias, next); err != nil {
		a.Log(fmt.Sprintf("ctl: save FAILED configs/%s.json: %v", alias, err))
		return err
	}
	a.presets[alias] = next
	a.Log(fmt.Sprintf("ctl: renamed preset %s/%s to %s", alias, oldName, newName))
	a.emitStateLocked()
	return nil
}

// RemovePreset deletes the named preset and persists the remainder.
func (a *App) RemovePreset(alias, name string) error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	list := a.presets[alias]
	kept := make([]schema.Preset, 0, len(list))
	found := false
	removed := -1
	for i, p := range list {
		if p.Name == name {
			found = true
			removed = i
			continue
		}
		kept = append(kept, p)
	}
	if !found {
		return fmt.Errorf("no preset %q for alias %s", name, alias)
	}
	if len(kept) == 0 {
		return fmt.Errorf("refusing to remove the last preset of %s (edit it instead)", alias)
	}
	sel := a.selPreset
	a.presets[alias] = kept
	if err := store.SaveAlias(settings.ConfigsDir(), alias, kept); err != nil {
		a.Log(fmt.Sprintf("ctl: save FAILED configs/%s.json: %v", alias, err))
		return err
	}
	if a.alias() == alias {
		if sel < 0 || sel >= len(list) {
			sel = 0
		}
		switch {
		case sel == removed:
			a.applyPreset(0)
		case sel > removed:
			// Keep the same preset applied; only its index moved down.
			a.selPreset = sel - 1
		}
	}
	a.Log(fmt.Sprintf("ctl: removed preset %s/%s", alias, name))
	a.emitStateLocked()
	return nil
}

// mergeConfig overlays non-zero fields of patch onto base (basic/advanced
// per-field, so a partial patch keeps the rest).
func mergeConfig(base, patch schema.Config) schema.Config {
	out := base
	if patch.Basic.Model != "" {
		out.Basic.Model = patch.Basic.Model
	}
	if patch.Basic.Alias != "" {
		out.Basic.Alias = patch.Basic.Alias
	}
	if patch.Basic.Host != "" {
		out.Basic.Host = patch.Basic.Host
	}
	if patch.Basic.Port != 0 {
		out.Basic.Port = patch.Basic.Port
	}
	if patch.Basic.Context != 0 {
		out.Basic.Context = patch.Basic.Context
	}
	if patch.Basic.GPULayers != "" {
		out.Basic.GPULayers = patch.Basic.GPULayers
	}
	if patch.Basic.ParallelSlots != 0 {
		out.Basic.ParallelSlots = patch.Basic.ParallelSlots
	}
	if patch.Basic.FlashAttention != "" {
		out.Basic.FlashAttention = patch.Basic.FlashAttention
	}
	if patch.Basic.KCache != "" {
		out.Basic.KCache = patch.Basic.KCache
	}
	if patch.Basic.VCache != "" {
		out.Basic.VCache = patch.Basic.VCache
	}
	if patch.Basic.Jinja {
		out.Basic.Jinja = true
	}
	if patch.Basic.ReasoningFmt != "" {
		out.Basic.ReasoningFmt = patch.Basic.ReasoningFmt
	}
	if patch.Basic.ReasoningBudget != 0 {
		out.Basic.ReasoningBudget = patch.Basic.ReasoningBudget
	}
	if patch.Advanced.SpecType != "" {
		out.Advanced.SpecType = patch.Advanced.SpecType
		out.Advanced.DraftNMax = patch.Advanced.DraftNMax
		out.Advanced.DraftNgl = patch.Advanced.DraftNgl
		out.Advanced.DraftTypeK = patch.Advanced.DraftTypeK
		out.Advanced.DraftTypeV = patch.Advanced.DraftTypeV
	}
	if patch.Advanced.MMProj != "" {
		out.Advanced.MMProj = patch.Advanced.MMProj
	}
	if patch.Advanced.NoMMProjOffload {
		out.Advanced.NoMMProjOffload = true
	}
	if patch.Advanced.Fit != "" {
		out.Advanced.Fit = patch.Advanced.Fit
	}
	if patch.Advanced.Perf {
		out.Advanced.Perf = true
	}
	if patch.Advanced.Metrics {
		out.Advanced.Metrics = true
	}
	if patch.Advanced.SleepIdle != 0 {
		out.Advanced.SleepIdle = patch.Advanced.SleepIdle
	}
	if patch.Advanced.CacheRAM != 0 {
		out.Advanced.CacheRAM = patch.Advanced.CacheRAM
	}
	if patch.Advanced.Mlock {
		out.Advanced.Mlock = true
	}
	if patch.Advanced.NoMmap {
		out.Advanced.NoMmap = true
	}
	if patch.Advanced.CPUThreads != 0 {
		out.Advanced.CPUThreads = patch.Advanced.CPUThreads
	}
	if patch.Advanced.UBatch != 0 {
		out.Advanced.UBatch = patch.Advanced.UBatch
	}
	if patch.Advanced.Batch != 0 {
		out.Advanced.Batch = patch.Advanced.Batch
	}
	if patch.Advanced.OverrideTensor != "" {
		out.Advanced.OverrideTensor = patch.Advanced.OverrideTensor
	}
	return out
}

func firstName(list []schema.Preset) string {
	if len(list) == 0 {
		return ""
	}
	return list[0].Name
}
