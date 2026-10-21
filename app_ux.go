package main

// app_ux.go — desktop/UX actions that do not fit the original controller:
// rescanning, deleting a model, favorites, the chat/web-UI link, native file
// pickers, restart and preset reset/export.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"llamaherd/control"
	"llamaherd/schema"
	"llamaherd/settings"
	"llamaherd/store"
)

// RescanModels re-walks the models directory, creates library stubs for new
// files and keeps the previous selection (by path and preset name) when it
// still exists.
func (a *App) RescanModels() error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	prevPath, prevPreset := "", ""
	if a.selModel >= 0 && a.selModel < len(a.models) {
		prevPath = a.models[a.selModel].Path
		if names := a.presetNames(); a.selPreset >= 0 && a.selPreset < len(names) {
			prevPreset = names[a.selPreset]
		}
	}
	a.initModels()
	a.initPresets()
	a.reloadBenchmarksLocked()
	a.selModel = -1
	if prevPath != "" {
		a.selModel = a.indexOfModelPath(prevPath)
	}
	if a.selModel < 0 {
		for i, m := range a.models {
			if !m.IsMMProj {
				a.selModel = i
				break
			}
		}
	}
	if a.selModel >= 0 {
		idx := 0
		if prevPreset != "" {
			idx = a.presetIndex(prevPreset)
		}
		a.applyPreset(idx)
	}
	a.dirty = false
	a.Log(fmt.Sprintf("models: rescanned — %d loadable model(s)", len(a.models)))
	a.emitStateLocked()
	return nil
}

// DeleteModel removes a model's .gguf from disk and its library metadata. The
// preset file is kept so re-downloading the model restores its settings.
func (a *App) DeleteModel(ref string) error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	i, err := a.modelIndex(ref)
	if err != nil {
		return err
	}
	m := a.models[i]
	if a.serverUsesPath(m.Path) {
		return fmt.Errorf("stop the running server before deleting %q", m.Display)
	}
	if err := os.Remove(m.Path); err != nil {
		return fmt.Errorf("delete %s: %w", m.Path, err)
	}
	_ = store.DeleteMeta(settings.LibraryDir(), m.ID)
	_ = store.DeleteModelBenchmarks(settings.BenchmarksDir(), m.ID)
	delete(a.presets, m.Alias)
	prevPath := ""
	if a.selModel >= 0 && a.selModel < len(a.models) && a.selModel != i {
		prevPath = a.models[a.selModel].Path
	}
	a.initModels()
	a.initPresets()
	a.reloadBenchmarksLocked()
	a.selModel = -1
	if prevPath != "" {
		a.selModel = a.indexOfModelPath(prevPath)
	}
	if a.selModel < 0 {
		for j, mm := range a.models {
			if !mm.IsMMProj {
				a.selModel = j
				break
			}
		}
	}
	if a.selModel >= 0 {
		a.applyPreset(0)
	}
	a.dirty = false
	a.Log(fmt.Sprintf("library: deleted %s (%s)", m.Display, m.Path))
	a.emitStateLocked()
	return nil
}

// SetModelFavorite pins/unpins a model in the list (stored in library meta).
func (a *App) SetModelFavorite(ref string, favorite bool) error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	i, err := a.modelIndex(ref)
	if err != nil {
		return err
	}
	m := &a.models[i]
	if err := store.UpdateMetaByID(settings.LibraryDir(), m.ID, m.Path, func(meta *store.Meta) {
		meta.Favorite = favorite
	}); err != nil {
		return err
	}
	m.Favorite = favorite
	if favorite {
		a.Log("library: pinned " + m.Display)
	} else {
		a.Log("library: unpinned " + m.Display)
	}
	a.emitStateLocked()
	return nil
}

// RevealModel opens the file manager at the model's directory.
func (a *App) RevealModel(ref string) error {
	a.stateMu.RLock()
	i, err := a.modelIndex(ref)
	var path string
	if err == nil {
		path = a.models[i].Path
	}
	a.stateMu.RUnlock()
	if err != nil {
		return err
	}
	return control.Reveal(path)
}

// ChatURL is the address of the running server's built-in web UI.
func (a *App) ChatURL() string {
	info := a.mgr.Info()
	if info.Listen != "" {
		return "http://" + info.Listen + "/"
	}
	a.stateMu.RLock()
	host, port := a.cfg.Basic.Host, a.cfg.Basic.Port
	a.stateMu.RUnlock()
	if host == "" {
		host = "127.0.0.1"
	}
	if port > 0 {
		return fmt.Sprintf("http://%s:%d/", host, port)
	}
	return ""
}

// OpenChat opens llama-server's built-in chat UI in the default browser.
func (a *App) OpenChat() error {
	info := a.mgr.Info()
	if info.Status != control.StatusRunning {
		return fmt.Errorf("start the server first")
	}
	if a.ctx == nil {
		return fmt.Errorf("chat is only available in the GUI")
	}
	return settings.OpenExternalURL(a.ChatURL())
}

// PickServerBinary opens a native file picker for llama-server.
func (a *App) PickServerBinary() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("file pickers are only available in the GUI")
	}
	a.stateMu.RLock()
	def := filepath.Dir(a.bin)
	a.stateMu.RUnlock()
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select the llama-server binary",
		DefaultDirectory: def,
	})
}

// PickModelsDir opens a native directory picker for the models folder.
func (a *App) PickModelsDir() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("file pickers are only available in the GUI")
	}
	a.stateMu.RLock()
	def := a.modelsDir
	a.stateMu.RUnlock()
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select the models directory",
		DefaultDirectory: def,
	})
}

// PickModelFiles opens a native multi-file picker for .gguf models (import).
func (a *App) PickModelFiles() ([]string, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("file pickers are only available in the GUI")
	}
	a.stateMu.RLock()
	def := a.modelsDir
	a.stateMu.RUnlock()
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select one or more GGUF models",
		DefaultDirectory: def,
		Filters:          []runtime.FileFilter{{DisplayName: "GGUF models", Pattern: "*.gguf"}},
	})
}

// PickModelFile opens a native picker for a .gguf sidecar (mmproj / draft).
func (a *App) PickModelFile(title string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("file pickers are only available in the GUI")
	}
	a.stateMu.RLock()
	def := a.modelsDir
	a.stateMu.RUnlock()
	if strings.TrimSpace(title) == "" {
		title = "Select a GGUF file"
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: def,
		Filters:          []runtime.FileFilter{{DisplayName: "GGUF models", Pattern: "*.gguf"}},
	})
}

// RestartServer stops the managed server (when it is running) and starts it
// again with the current form values.
func (a *App) RestartServer() error {
	info := a.mgr.Info()
	if info.Status == control.StatusStopped || info.Status == control.StatusCrashed {
		a.StartServer()
		return a.startWarning()
	}
	if info.External {
		return fmt.Errorf("refusing to restart: server was started outside LlamaHerd")
	}
	a.Log("restarting llama-server …")
	if err := a.mgr.Stop(); err != nil {
		return err
	}
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		st := a.mgr.Info().Status
		if st == control.StatusStopped || st == control.StatusCrashed {
			a.StartServer()
			return a.startWarning()
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("server did not stop within 12s; use Force stop")
}

// startWarning returns the last pre-flight warning, if any, as an error.
func (a *App) startWarning() error {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	if a.warn == "" {
		return nil
	}
	return fmt.Errorf("%s", a.warn)
}

// AutoFitConfig chooses the largest context and GPU-layer count that pass the
// VRAM guard on this machine, applies them to the form and returns a summary.
func (a *App) AutoFitConfig() (string, error) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if a.cfg.Basic.Model == "" {
		return "", fmt.Errorf("select a model first")
	}
	fitted := control.FitPreset(a.cfg, maxInt(51200, a.cfg.Basic.Context))
	a.cfg = fitted
	a.dirty = true
	a.warn = ""
	msg := fmt.Sprintf("fitted: %dk context, -ngl %s", fitted.Basic.Context/1024, fitted.Basic.GPULayers)
	a.Log(msg)
	a.emitStateLocked()
	return msg, nil
}

// ResetPreset replaces a preset with the synthesized, auto-fitted default for
// the selected model.
func (a *App) ResetPreset(name string) error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	alias := a.alias()
	if alias == "" {
		return fmt.Errorf("select a model first")
	}
	list := a.presets[alias]
	if len(list) == 0 {
		return fmt.Errorf("no presets for alias %s", alias)
	}
	idx := 0
	if name != "" {
		idx = -1
		for i := range list {
			if list[i].Name == name {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("no preset %q for alias %s", name, alias)
		}
	}
	modelPath := a.cfg.Basic.Model
	if a.selModel >= 0 && a.selModel < len(a.models) {
		modelPath = a.models[a.selModel].Path
	}
	p := defaultPreset(alias, modelPath)
	p.Name = list[idx].Name
	next := append([]schema.Preset(nil), list...)
	next[idx] = p
	if err := store.SaveAlias(settings.ConfigsDir(), alias, next); err != nil {
		return err
	}
	a.presets[alias] = next
	a.selPreset = idx
	a.cfg = p.AsConfig()
	a.dirty = false
	a.Log(fmt.Sprintf("preset: reset %s/%s to the model default", alias, p.Name))
	a.emitStateLocked()
	return nil
}

// ExportPresets returns the current alias's portable preset JSON.
func (a *App) ExportPresets() (string, error) {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	alias := a.alias()
	if alias == "" {
		return "", fmt.Errorf("select a model first")
	}
	data, err := store.AliasJSON(alias, a.presets[alias])
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// serverUsesPath reports whether the managed server was launched from path.
func (a *App) serverUsesPath(path string) bool {
	st := a.mgr.Info().Status
	if st == control.StatusStopped || st == control.StatusCrashed {
		return false
	}
	return a.runCfg != nil && settings.SamePath(a.runCfg.Basic.Model, path)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
