package main

// app_last_run.go — remember the model/preset the user last selected or started
// and restore it on the next launch (never auto-starts the server).

import (
	"strings"
	"time"

	"llamaherd/settings"
)

// rememberSelectionLocked persists the current model/preset. Caller holds stateMu.
func (a *App) rememberSelectionLocked() {
	alias := a.alias()
	if alias == "" {
		return
	}
	preset := ""
	if names := a.presetNames(); a.selPreset >= 0 && a.selPreset < len(names) {
		preset = names[a.selPreset]
	}
	if err := settings.SaveLastRun(settings.LastRun{Alias: alias, Preset: preset, At: time.Now().Unix()}); err != nil {
		a.Log("state: could not save the last selection: " + err.Error())
	}
}

// rememberRun persists the model/preset that successfully started.
func (a *App) rememberRun(alias, preset string) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return
	}
	if err := settings.SaveLastRun(settings.LastRun{Alias: alias, Preset: preset, At: time.Now().Unix()}); err != nil {
		a.Log("state: could not save the last run: " + err.Error())
	}
}

// restoreLastRun selects the remembered model/preset when both still exist.
// It leaves the default selection untouched when the model was deleted/renamed.
func (a *App) restoreLastRun() bool {
	last, ok := settings.LoadLastRun()
	if !ok {
		return false
	}
	idx := -1
	for i, m := range a.models {
		if !m.IsMMProj && strings.EqualFold(m.Alias, last.Alias) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	a.selModel = idx
	list := a.presets[a.models[idx].Alias]
	if len(list) == 0 {
		return true
	}
	preset := 0
	for i, p := range list {
		if strings.EqualFold(p.Name, last.Preset) {
			preset = i
			break
		}
	}
	a.selPreset = preset
	a.cfg = list[preset].AsConfig()
	a.Log("startup: restored last selection " + a.models[idx].Alias + "/" + list[preset].Name)
	return true
}
