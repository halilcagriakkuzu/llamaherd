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
	return a.models
}

// Presets returns the named presets of an alias (api.Control).
func (a *App) Presets(alias string) []schema.Preset {
	return a.presetList(alias)
}

// SelectModelByAlias selects the loadable model whose alias equals the given
// string or uniquely contains it as a substring (e.g. "27b-local" vs
// "-uncensored-27b-local"). Ambiguity is an error; mmproj entries never match.
func (a *App) SelectModelByAlias(alias string) error {
	if alias == "" {
		return nil // keep current selection
	}
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
		a.SelectModel(exact)
		return nil
	case count == 1:
		a.SelectModel(fuzzy)
		return nil
	case count == 0:
		return fmt.Errorf("no model alias matching %q (use `llamaherd list`)", alias)
	default:
		return fmt.Errorf("alias %q is ambiguous (%d matches); use a longer string", alias, count)
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
		if a.findPreset(a.alias(), preset) == nil {
			return fmt.Errorf("no preset %q for alias %s (use `llamaherd list %s`)", preset, a.alias(), a.alias())
		}
		a.applyPreset(a.presetIndex(preset))
		a.dirty = false
		a.Log(fmt.Sprintf("ctl: preset %s/%s", a.alias(), preset))
	}
	a.Log(fmt.Sprintf("ctl: open %s [%s]", a.alias(), a.presetNames()[a.selPreset]))
	a.StartServer()
	if a.warn != "" {
		// StartServer refused (e.g. VRAM pre-flight): surface it to the caller.
		return fmt.Errorf("%s", a.warn)
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
	a.emitState()
	return nil
}

// UpdatePreset replaces the named preset's config and persists it.
func (a *App) UpdatePreset(alias, name string, cfg schema.Config) error {
	list := a.presets[alias]
	for i := range list {
		if list[i].Name != name {
			continue
		}
		list[i] = cfg.AsPreset(name)
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
		a.emitState()
		return nil
	}
	return fmt.Errorf("no preset %q for alias %s", name, alias)
}

// RemovePreset deletes the named preset and persists the remainder.
func (a *App) RemovePreset(alias, name string) error {
	list := a.presets[alias]
	kept := make([]schema.Preset, 0, len(list))
	found := false
	for _, p := range list {
		if p.Name == name {
			found = true
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
	a.presets[alias] = kept
	if err := store.SaveAlias(settings.ConfigsDir(), alias, kept); err != nil {
		a.Log(fmt.Sprintf("ctl: save FAILED configs/%s.json: %v", alias, err))
		return err
	}
	if a.alias() == alias && a.presetNames()[a.selPreset] == name {
		a.applyPreset(0)
	}
	a.Log(fmt.Sprintf("ctl: removed preset %s/%s", alias, name))
	a.emitState()
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
