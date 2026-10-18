package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"llamaherd/api"
	"llamaherd/control"
	"llamaherd/doctor"
	"llamaherd/schema"
	"llamaherd/settings"
	"llamaherd/store"
)

// Snapshot is the complete UI state. It is sent to the frontend on every
// change (event "state") and on GetState().
type Snapshot struct {
	Status         string         `json:"status"` // starting | stopping | running | stopped | crashed
	CrashExit      int            `json:"crashExit"`
	Server         schema.Server  `json:"server"`
	Models         []schema.Model `json:"models"`
	Categories     [][]string     `json:"categories"` // persisted category folders (may be empty)
	SelectedModel  int            `json:"selectedModel"`
	Alias          string         `json:"alias"`
	PresetNames    []string       `json:"presetNames"`
	SelectedPreset int            `json:"selectedPreset"`
	Config         schema.Config  `json:"config"`
	ConfigPath     string         `json:"configPath"`
	Dirty          bool           `json:"dirty"`
	CommandLine    string         `json:"commandLine"`
	BootLines      []string       `json:"bootLines"`
	// Warning is a blocking pre-flight message (currently: the VRAM guard).
	// Non-empty => the last Start was refused; the UI shows it as a banner.
	Warning string `json:"warning"`
	// VramEstimate is the human-readable predicted GPU footprint.
	VramEstimate string `json:"vramEstimate"`
	// LlamaUpdating is true while scripts/update-llamacpp.sh runs.
	LlamaUpdating bool `json:"llamaUpdating"`
	// FirstRun is true when no user config file exists and the llama-server
	// binary or models directory is missing: the UI shows a setup prompt
	// instead of failing later.
	FirstRun       bool   `json:"firstRun"`
	ServerBin      string `json:"serverBin"`
	ModelsDir      string `json:"modelsDir"`
	UserConfigPath string `json:"userConfigPath"`
	// Doctor is the last environment report (see the doctor package); the UI
	// shows it as a first-run panel/banner.
	Doctor doctor.Report `json:"doctor"`
	// Download is the live Hugging Face model-download state.
	Download DownloadState `json:"download"`
}

// App is the Wails-bound application. Start/stop control the real llama-server
// process (control.Manager); model scanning and configs/<alias>.json
// persistence are handled here.
type App struct {
	ctx       context.Context
	mgr       *control.Manager
	bin       string // resolved llama-server binary (env / config / discovery)
	modelsDir string // resolved models directory (env / config / default)
	firstRun  bool   // true => show the setup prompt
	models    []schema.Model
	cats      [][]string // persisted (possibly empty) category folders
	selModel  int
	selPreset int
	presets   map[string][]schema.Preset
	cfg       schema.Config
	dirty     bool
	bootLines []string
	eventCh   chan eventMsg
	warn      string // blocking pre-flight message (VRAM guard), "" = none
	llama     llamaUpdate
	doctorMu  sync.Mutex
	doctorRep doctor.Report
	dlMu      sync.Mutex
	dl        DownloadState
	dlCancel  context.CancelFunc

	// runCfg is the config the CURRENTLY loaded/running server was launched
	// with. The Status panel reports this, not the (possibly different)
	// model/preset the user has selected in the form. nil = nothing loaded.
	runCfg *schema.Config
}

// eventMsg serializes every frontend emission through the pump goroutine —
// runtime.EventsEmit is called ONLY from there: HTTP-handler goroutines calling
// Emit directly crash the Wails runtime.
type eventMsg struct {
	kind int // 0 = state, 1 = log line
	line string
	snap Snapshot
}

// NewApp creates the App.
func NewApp() *App {
	bin := settings.ServerBin()
	modelsDir := settings.ModelsDir()
	return &App{
		bin:       bin,
		modelsDir: modelsDir,
		firstRun:  computeFirstRun(bin, modelsDir),
		mgr:       control.NewManager(bin),
		presets:   map[string][]schema.Preset{},
		eventCh:   make(chan eventMsg, 1024),
	}
}

// computeFirstRun reports whether the setup prompt should be shown: no user
// config file yet AND the binary or models directory cannot be found.
func computeFirstRun(bin, modelsDir string) bool {
	if settings.ConfigExists() {
		return false
	}
	if !control.BinaryExists(bin) {
		return true
	}
	if ents, err := os.ReadDir(modelsDir); err != nil || len(ents) == 0 {
		return true
	}
	return false
}

// SetPaths saves the llama-server binary and models directory to the user
// config file and applies them live (no restart). Bound to the first-run
// setup prompt.
func (a *App) SetPaths(serverBin, modelsDir string) error {
	if strings.TrimSpace(serverBin) == "" && strings.TrimSpace(modelsDir) == "" {
		return fmt.Errorf("nothing to save: provide the llama-server path and/or the models directory")
	}
	f := settings.Config()
	if s := strings.TrimSpace(serverBin); s != "" {
		f.ServerBin = s
		// Let the checkout/build dirs re-derive from the new binary.
		f.LlamaDir = ""
		f.LlamaBuildDir = ""
	}
	if s := strings.TrimSpace(modelsDir); s != "" {
		f.ModelsDir = s
	}
	if err := settings.Save(f); err != nil {
		return err
	}
	settings.Reload()
	a.reloadSettings()
	a.Log(fmt.Sprintf("settings: saved to %s", settings.ConfigFile()))
	a.emitState()
	return nil
}

// reloadSettings re-resolves config after the user config file changed and
// rebuilds the model/preset state against it.
func (a *App) reloadSettings() {
	a.bin = settings.ServerBin()
	a.modelsDir = settings.ModelsDir()

	// Rebind the manager to the (possibly new) binary.
	a.mgr = control.NewManager(a.bin)
	a.mgr.SetCallbacks(
		func() { a.emitState() },
		func(line string) { a.Log(line) },
	)

	a.bootLines = append(a.bootLines,
		"settings: llama-server = "+a.bin,
		"settings: models dir = "+a.modelsDir)

	a.initModels()
	a.initPresets()
	a.selModel = -1
	for i, m := range a.models {
		if !m.IsMMProj {
			a.selModel = i
			break
		}
	}
	if p := a.presetList(a.alias()); len(p) > 0 {
		a.cfg = p[0].AsConfig()
	}
	a.selPreset = 0
	a.runCfg = nil
	a.firstRun = computeFirstRun(a.bin, a.modelsDir)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.pumpEvents()
	a.mgr.SetCallbacks(
		func() { a.emitState() },
		func(line string) { a.Log(line) },
	)

	a.initBackend()
	a.initModels()
	a.initPresets()

	// Select the first loadable model, apply its first preset, then check
	// whether a server is ALREADY listening (external).
	a.selModel = -1
	if a.selModel < 0 {
		for i, m := range a.models {
			if !m.IsMMProj {
				a.selModel = i
				break
			}
		}
	}
	if p := a.presetList(a.alias()); len(p) > 0 {
		a.cfg = p[0].AsConfig()
	}
	a.selPreset = 0

	a.mgr.AttachExternal(a.cfg.Basic.Host, a.cfg.Basic.Port)
	if a.mgr.Info().Status == control.StatusRunning {
		rc := a.cfg
		a.runCfg = &rc
	}

	// Serve the control API so CLI/AI agents drive the same App.
	go func() {
		if err := api.Serve(a); err != nil {
			a.Log("ctl api: " + err.Error())
		}
	}()

	// Environment report for the first-run panel/banner (cached in snapshot).
	go func() {
		a.refreshDoctor()
		a.emitState()
	}()

	a.emitState()
}

// initBackend checks the configured binary at boot (existence only logged).
func (a *App) initBackend() {
	if !control.BinaryExists(a.bin) {
		a.bootLines = append(a.bootLines,
			"WARNING: llama-server binary missing: "+a.bin)
	} else {
		a.bootLines = append(a.bootLines, "backend: "+a.bin)
	}
	if a.firstRun {
		a.bootLines = append(a.bootLines,
			"first run: set the llama-server binary and models directory")
	}
}

// initModels scans the models directory for *.gguf files.
func (a *App) initModels() {
	modelsDir := a.modelsDir
	if modelsDir == "" {
		modelsDir = settings.ModelsDir()
	}
	if settings.IsDefaultModelsDir() {
		// Create the default folder so first-run users have somewhere to drop
		// models (an explicitly configured path is never created for them).
		_ = os.MkdirAll(modelsDir, 0o755)
	}
	scanned := store.ScanModels(modelsDir)
	a.models = scanned
	if n := store.ApplyLibrary(settings.LibraryDir(), a.models); n > 0 {
		a.bootLines = append(a.bootLines,
			fmt.Sprintf("library: created %d uncategorized meta stub(s)", n))
	}
	a.cats = store.LoadCategories(settings.LibraryDir())
	a.bootLines = append(a.bootLines,
		fmt.Sprintf("models: scanned %d *.gguf under %s", len(a.models), modelsDir))
}

// initPresets synthesizes a default preset for every scanned model, then
// overrides it from disk (configs/<alias>.json) when present.
func (a *App) initPresets() {
	for i := range a.models {
		if a.models[i].IsMMProj {
			continue
		}
		a.models[i].Alias = a.aliasForPath(a.models[i].Path)
	}
	for i := range a.models {
		m := a.models[i]
		if m.IsMMProj {
			continue
		}
		if _, ok := a.presets[m.Alias]; !ok {
			a.presets[m.Alias] = []schema.Preset{defaultPreset(m.Alias, m.Path)}
			// New library model: persist its hardware-fitted default so it is
			// an editable configs/<alias>.json from the start.
			if _, err := os.Stat(filepath.Join(settings.ConfigsDir(), m.Alias+".json")); err != nil {
				if store.SaveAlias(settings.ConfigsDir(), m.Alias, a.presets[m.Alias]) == nil {
					a.bootLines = append(a.bootLines,
						fmt.Sprintf("config: seeded configs/%s.json", m.Alias))
				}
			}
		}
		if f, err := store.LoadAlias(settings.ConfigsDir(), m.Alias); err == nil {
			a.presets[m.Alias] = f.Presets
			a.bootLines = append(a.bootLines,
				fmt.Sprintf("config: loaded %d preset(s) from configs/%s.json", len(f.Presets), m.Alias))
		}
	}
}

// aliasForPath returns the file-name-derived alias for a model path.
func (a *App) aliasForPath(path string) string {
	return store.AliasFor(settings.LibraryDir(), path)
}

func (a *App) indexOfModelPath(path string) int {
	for i, m := range a.models {
		if m.Path == path {
			return i
		}
	}
	return -1
}

func defaultPreset(alias, modelPath string) schema.Preset {
	p := schema.Preset{
		Name: "default",
		Basic: schema.Basic{
			Model:          modelPath,
			Alias:          alias,
			Host:           "127.0.0.1",
			Port:           8090,
			Context:        32768,
			GPULayers:      "all",
			ParallelSlots:  1,
			FlashAttention: "on",
			KCache:         "q8_0",
			VCache:         "q8_0",
			Jinja:          true,
			ReasoningFmt:   "deepseek",
		},
		Advanced: schema.Advanced{Fit: "off", SleepIdle: 300},
	}
	// Keep the synthesized preset inside the GPU budget: if "all" does not fit,
	// use the largest -ngl that does (otherwise the VRAM guard refuses to
	// start it).
	if ngl := control.SafeGpuLayers(p.AsConfig()); ngl >= 0 {
		p.Basic.GPULayers = strconv.Itoa(ngl)
	}
	return p
}

// GetState returns the current snapshot (frontend calls this once at boot).
func (a *App) GetState() Snapshot {
	return a.snapshot()
}

// SelectModel selects a model list row. mmproj rows fill the --mmproj field
// and do NOT change the model/config selection.
func (a *App) SelectModel(index int) {
	if index < 0 || index >= len(a.models) {
		return
	}
	m := a.models[index]
	if m.IsMMProj {
		a.cfg.Advanced.MMProj = m.Path
		a.dirty = true
		a.Log(fmt.Sprintf("mmproj: %s", m.Display))
	} else {
		a.selModel = index
		a.applyPreset(0)
		a.dirty = false
		a.warn = ""
	}
	a.emitState()
}

// SelectPreset applies a named preset to the shared Config.
func (a *App) SelectPreset(name string) {
	p := a.findPreset(a.alias(), name)
	if p == nil {
		return
	}
	a.applyPreset(a.presetIndex(name))
	a.dirty = false
	a.warn = ""
	a.Log(fmt.Sprintf("preset: %s/%s", a.alias(), name))
	a.emitState()
}

// UpdateConfig stores the (already collected) form values in the shared
// Config. Saves/loads/preset switches clear the dirty flag; plain edits set it.
func (a *App) UpdateConfig(cfg schema.Config) {
	a.cfg = cfg
	a.dirty = true
	a.warn = "" // config changed: any earlier block no longer applies
	a.emitState()
}

// SaveConfig writes the current config as the current preset name into
// configs/<alias>.json, upserting by preset name.
func (a *App) SaveConfig() {
	alias := a.alias()
	names := a.presetNames()
	name := names[a.selPreset]

	list := a.presets[alias]
	kept := make([]schema.Preset, 0, len(list)+1)
	replaced := false
	for _, p := range list {
		if p.Name == name {
			kept = append(kept, a.cfg.AsPreset(name))
			replaced = true
		} else {
			kept = append(kept, p)
		}
	}
	if !replaced {
		kept = append(kept, a.cfg.AsPreset(name))
	}

	if err := store.SaveAlias(settings.ConfigsDir(), alias, kept); err != nil {
		a.Log(fmt.Sprintf("save FAILED configs/%s.json: %v", alias, err))
		a.emitState()
		return
	}
	a.presets[alias] = kept
	a.dirty = false
	a.Log(fmt.Sprintf("saved configs/%s.json [preset %s]", alias, name))
	a.emitState()
}

// LoadConfig reads configs/<alias>.json from disk and replaces the alias's
// in-memory presets. Missing/malformed file: logged, nothing changed.
func (a *App) LoadConfig() {
	alias := a.alias()
	f, err := store.LoadAlias(settings.ConfigsDir(), alias)
	if err != nil {
		a.Log(fmt.Sprintf("loaded configs/%s.json FAILED: %v (nothing changed)", alias, err))
		a.emitState()
		return
	}
	a.presets[alias] = f.Presets
	name := a.presetNames()[a.selPreset]
	idx := 0
	for i, p := range f.Presets {
		if p.Name == name {
			idx = i
			break
		}
	}
	a.applyPreset(idx)
	a.selPreset = idx
	a.dirty = false
	a.Log(fmt.Sprintf("loaded configs/%s.json [%d preset(s), preset %s]",
		alias, len(f.Presets), f.Presets[idx].Name))
	a.emitState()
}

// StartServer launches the real llama-server. Errors (port busy,
// missing binary) are logged and leave the state machine untouched. Before
// spawning anything it runs the VRAM pre-flight: a launch that would exceed
// 90% of the GPU's memory is refused outright (that runaway allocation is what
// previously OOM-killed the whole WSL distro).
func (a *App) StartServer() {
	est := control.EstimateVram(a.cfg)
	if est.Warn != "" {
		a.warn = est.Warn
		a.Log("start BLOCKED — " + est.Warn)
		a.Log("start BLOCKED — " + est.Detail)
		a.emitState()
		return
	}
	a.warn = ""
	a.Log(est.Detail)
	a.Log("starting llama-server: " + a.commandLine())
	if err := a.mgr.Start(a.cfg); err != nil {
		a.Log(fmt.Sprintf("start FAILED: %v", err))
		a.emitState()
		return
	}
	// Freeze the launched config: the Status panel reports THIS, even if the
	// user later picks another model/preset in the form.
	rc := a.cfg
	a.runCfg = &rc
	a.emitState()
}

// StopServer terminates the managed process (SIGTERM, SIGKILL after 3s).
func (a *App) StopServer() {
	if err := a.mgr.Stop(); err != nil {
		a.Log(fmt.Sprintf("stop FAILED: %v", err))
	}
	a.emitState()
}

// ----------------------------------------------------------------- internals

func (a *App) alias() string {
	if a.selModel < 0 || a.selModel >= len(a.models) {
		return ""
	}
	return a.models[a.selModel].Alias
}

func (a *App) presetList(alias string) []schema.Preset {
	return a.presets[alias]
}

func (a *App) presetNames() []string {
	names := []string{}
	for _, p := range a.presetList(a.alias()) {
		names = append(names, p.Name)
	}
	return names
}

func (a *App) findPreset(alias, name string) *schema.Preset {
	list := a.presetList(alias)
	for i := range list {
		if list[i].Name == name {
			return &list[i]
		}
	}
	return nil
}

func (a *App) presetIndex(name string) int {
	for i, n := range a.presetNames() {
		if n == name {
			return i
		}
	}
	return 0
}

func (a *App) applyPreset(index int) {
	list := a.presetList(a.alias())
	if index < 0 || index >= len(list) {
		index = 0
	}
	a.selPreset = index
	if len(list) > 0 {
		a.cfg = list[index].AsConfig()
	}
}

func (a *App) commandLine() string {
	return "llama-server " + strings.Join(control.BuildArgs(a.cfg), " ")
}

func (a *App) snapshot() Snapshot {
	info := a.mgr.Info()
	switch info.Status {
	case control.StatusStopped, control.StatusCrashed:
		a.runCfg = nil // nothing is loaded anymore
	}
	sp := a.mgr.Speed()
	if info.Status != control.StatusRunning {
		sp = control.SpeedStats{} // only the live server's speed is shown
	}
	srv := schema.Server{PID: "—", Uptime: "00:00:00", VRAM: "—", CPURAM: "—", TokPerSec: sp.Summary(), TokPerSecTip: sp.Tooltip()}
	srv.Status = string(info.Status)
	srv.Listen = info.Listen
	srv.Binary = a.bin
	// The WSL IP row is meaningful only inside WSL (the address the Windows
	// host uses to reach the distro); elsewhere it stays blank ("—").
	if settings.IsWSL() {
		srv.WSLIP = settings.LocalIPv4()
	}
	// Model / context / GPU layers describe the LOADED server, never the
	// currently selected form values. When nothing is loaded these stay "—".
	srv.Model = "—"
	srv.Context = "—"
	srv.GPULayers = "—"
	if rc := a.runCfg; rc != nil {
		if rc.Basic.Model != "" {
			srv.Model = filepath.Base(rc.Basic.Model)
		}
		srv.Context = fmt.Sprint(rc.Basic.Context)
		srv.GPULayers = rc.Basic.GPULayers
	}
	// The real serving context from /props wins over the configured value.
	if info.NCtx > 0 {
		srv.Context = fmt.Sprint(info.NCtx)
	}
	switch info.Status {
	case control.StatusRunning:
		srv.PID = "—"
		if info.PID > 0 {
			srv.PID = fmt.Sprint(info.PID)
		}
		if !info.ReadyAt.IsZero() {
			srv.Uptime = uptime(info.ReadyAt)
		} else {
			srv.Uptime = "00:00:00"
		}
	case control.StatusStarting, control.StatusStopping:
		srv.PID = "—"
		srv.Uptime = "00:00:00"
	case control.StatusStopped, control.StatusCrashed:
		srv.PID = "—"
		srv.Uptime = "00:00:00"
	}
	// VRAM / CPU RAM rows: live numbers from nvidia-smi + /proc.
	srv.VRAM = control.HumanMB(info.VRAMMB)
	srv.CPURAM = control.HumanMB(info.RAMMB)
	s := Snapshot{
		Status:         string(info.Status),
		CrashExit:      info.Exit,
		Server:         srv,
		Models:         a.models,
		Categories:     a.cats,
		SelectedModel:  a.selModel,
		Alias:          a.alias(),
		PresetNames:    a.presetNames(),
		SelectedPreset: a.selPreset,
		Config:         a.cfg,
		ConfigPath:     fmt.Sprintf("configs/%s.json", a.alias()),
		Dirty:          a.dirty,
		CommandLine:    a.commandLine(),
		BootLines:      a.bootLines,
		Warning:        a.warn,
		VramEstimate:   control.EstimateVram(a.cfg).Detail,
	}
	s.LlamaUpdating, _ = a.llamaState()
	s.FirstRun = a.firstRun
	s.ServerBin = a.bin
	s.ModelsDir = a.modelsDir
	s.UserConfigPath = settings.ConfigFile()
	s.Doctor = a.DoctorReport()
	s.Download = a.DownloadSnapshot()
	return s
}

// uptime formats a running server's uptime as hh:mm:ss.
func uptime(readyAt time.Time) string {
	d := time.Since(readyAt).Truncate(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (a *App) emitState() {
	s := a.snapshot()
	if a.ctx == nil {
		// Headless (daemon) mode: no frontend to notify; print a compact
		// status line for CLI observers.
		fmt.Printf("[state] status=%s model=%s preset=%s/%d dirty=%v\n",
			s.Status, s.Alias, firstPresetName(s.PresetNames), s.SelectedPreset, s.Dirty)
		return
	}
	a.queue(eventMsg{kind: 0, snap: s})
}

func firstPresetName(names []string) string {
	if len(names) == 0 {
		return "—"
	}
	return names[0]
}

// Log appends a line to the log console and pushes it to the frontend.
func (a *App) Log(line string) {
	if a.ctx == nil {
		fmt.Println("[log] " + line)
		return
	}
	a.queue(eventMsg{kind: 1, line: line})
}

// queue hands an event to the pump. It drops the event rather than blocking
// when the buffer is full.
func (a *App) queue(e eventMsg) {
	select {
	case a.eventCh <- e:
	default:
	}
}

// pumpEvents is the ONLY goroutine allowed to call runtime.EventsEmit.
func (a *App) pumpEvents() {
	for e := range a.eventCh {
		if a.ctx == nil {
			return
		}
		switch e.kind {
		case 0:
			runtime.EventsEmit(a.ctx, "state", e.snap)
		case 1:
			runtime.EventsEmit(a.ctx, "log",
				time.Now().Format("15:04:05")+" "+e.line)
		}
	}
}

// startupHeadless initialises the same core as startup() without any Wails
// runtime (used by `llamaherd daemon`).
func (a *App) startupHeadless() {
	a.mgr.SetCallbacks(
		func() { a.emitState() },
		func(line string) { a.Log(line) },
	)
	a.initBackend()
	a.initModels()
	a.initPresets()
	a.selModel = -1
	if a.selModel < 0 {
		for i, m := range a.models {
			if !m.IsMMProj {
				a.selModel = i
				break
			}
		}
	}
	if p := a.presetList(a.alias()); len(p) > 0 {
		a.cfg = p[0].AsConfig()
	}
	a.selPreset = 0
	a.mgr.AttachExternal(a.cfg.Basic.Host, a.cfg.Basic.Port)
	if a.mgr.Info().Status == control.StatusRunning {
		rc := a.cfg
		a.runCfg = &rc
	}
	a.refreshDoctor()
	a.emitState()
}
