package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"llamaherd/api"
	"llamaherd/control"
	"llamaherd/doctor"
	"llamaherd/hf"
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
	// VramPercent / VramFits expose the same estimate as numbers, for the bar
	// and inline validation without parsing the detail string.
	VramPercent float64 `json:"vramPercent"`
	VramFits    bool    `json:"vramFits"`
	// Validation is the live form check (errors block Start, warnings do not).
	Validation []schema.Issue `json:"validation"`
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
	// Metrics is a 10-minute ring buffer of live samples (newest last).
	Metrics []schema.MetricsPoint `json:"metrics"`
	// Benchmarks are stored (model, config) speed results, fastest first.
	Benchmarks []schema.Benchmark `json:"benchmarks"`
	// Benchmarking is true while a probe is running.
	Benchmarking bool `json:"benchmarking"`
	// ConfigFingerprint identifies the form config; RunFingerprint identifies
	// the loaded server's config. A stored result is current only when both
	// match (same model/preset, unchanged fields).
	ConfigFingerprint string `json:"configFingerprint"`
	RunFingerprint    string `json:"runFingerprint"`
	// RecentLog is the server's own output tail (crash / not-responding hints).
	RecentLog []string `json:"recentLog"`
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
	dlWorker  bool // one queue pump goroutine at a time
	dlPausing bool // the current transfer was cancelled by PauseDownload
	dlSeq     int64
	dlLast    QueuedDownload // last started job, for RetryDownload

	// Host OS notifications (T14): notifier is injectable for tests; notifyStatus
	// remembers the last server state so transitions fire once.
	notifier     func(title, body string) error
	notifyMu     sync.Mutex
	notifyStatus string

	// metricHistory is a 10-minute ring of live samples; metricsAt throttles
	// sampling to the 5 s metrics tick.
	metricHistory []schema.MetricsPoint
	metricsAt     time.Time

	// bench is the stored benchmark table; benchmarking guards one probe.
	bench        []schema.Benchmark
	benchmarking bool

	// importMu allows only one local model import at a time.
	importMu sync.Mutex

	// logMu guards the in-memory log ring and the rotating file. Log() may be
	// called from any goroutine, including manager callbacks.
	logMu     sync.Mutex
	logRing   []logEntry
	logSeq    int64
	logPath   string
	logFile   *os.File
	logBytes  int64
	logFailed bool

	// stateMu guards every mutable field above that ends up in a Snapshot
	// (cfg, models, presets, selection, dirty, warn, runCfg, bootLines, ...).
	// Mutators hold the write lock, snapshot() takes the read lock, and
	// emitStateLocked is only called while the write lock is held.
	stateMu sync.RWMutex

	// runCfg is the config the CURRENTLY loaded/running server was launched
	// with. The Status panel reports this, not the (possibly different)
	// model/preset the user has selected in the form. nil = nothing loaded.
	runCfg *schema.Config
	// runPreset is the preset name that runCfg came from (benchmark labels).
	runPreset string
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
		notifier:  control.Notify,
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
	if s := strings.TrimSpace(serverBin); s != "" {
		if _, err := control.ProbeBinary(s); err != nil {
			return fmt.Errorf("llama-server check failed: %w", err)
		}
	}
	switch st := a.mgr.Info().Status; st {
	case control.StatusStarting, control.StatusRunning, control.StatusStopping, control.StatusFailed:
		return fmt.Errorf("stop the running server before changing paths")
	}
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
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
	if gen, err := settings.EnsureAPIToken(); err != nil {
		a.Log("ctl api: could not generate a token: " + err.Error())
	} else if gen {
		a.Log("ctl api: generated an access token (stored in the user config)")
	}
	settings.Reload()
	a.resetLog()
	a.reloadSettingsLocked()
	a.Log(fmt.Sprintf("settings: saved to %s", settings.ConfigFile()))
	a.emitStateLocked()
	return nil
}

// reloadSettingsLocked re-resolves config after the user config file changed
// and rebuilds the model/preset state against it. Caller holds stateMu.
func (a *App) reloadSettingsLocked() {
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
	a.runPreset = ""
	a.reloadBenchmarksLocked()
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
	a.initCatalog()
	a.stateMu.Lock()
	a.reloadBenchmarksLocked()
	a.stateMu.Unlock()

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
	a.restoreLastRun()

	a.mgr.AttachExternal(a.cfg.Basic.Host, a.cfg.Basic.Port)
	if a.mgr.Info().Status == control.StatusRunning {
		rc := a.cfg
		a.runCfg = &rc
		if names := a.presetNames(); a.selPreset >= 0 && a.selPreset < len(names) {
			a.runPreset = names[a.selPreset]
		}
	}

	// Serve the control API so CLI/AI agents drive the same App.
	a.ensureAPIToken()
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
	// ApplyLibrary assigns each row a stable id (and its alias); it must see the
	// same slice the app keeps so the ids stay consistent.
	if n := store.ApplyLibrary(settings.LibraryDir(), scanned); n > 0 {
		a.bootLines = append(a.bootLines,
			fmt.Sprintf("library: created %d uncategorized meta stub(s)", n))
	}
	for i := range scanned {
		if p := scanned[i].Provenance; p != nil && p.URL == "" {
			p.URL = hf.RepoURL(p.Repo)
		}
	}
	a.models = scanned
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

func (a *App) indexOfModelPath(path string) int {
	for i, m := range a.models {
		if settings.SamePath(m.Path, path) {
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
		Advanced: schema.Advanced{Fit: "off", Metrics: true, SleepIdle: 300},
	}
	// Keep the synthesized preset inside the GPU budget: choose a context that
	// fits and then the largest -ngl that still fits (otherwise the guard
	// refuses to start it).
	return control.FitPreset(p.AsConfig(), 32768).AsPreset("default")
}

// GetState returns the current snapshot (frontend calls this once at boot).
func (a *App) GetState() Snapshot {
	return a.snapshot()
}

// SelectModel selects a model list row. mmproj rows fill the --mmproj field
// and do NOT change the model/config selection.
func (a *App) SelectModel(index int) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
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
		a.markUsedLocked(index)
		a.rememberSelectionLocked()
	}
	a.emitStateLocked()
}

// markUsedLocked records the last selection time in library metadata.
func (a *App) markUsedLocked(index int) {
	if index < 0 || index >= len(a.models) {
		return
	}
	m := &a.models[index]
	if m.ID == "" {
		return
	}
	now := time.Now().Unix()
	_ = store.UpdateMetaByID(settings.LibraryDir(), m.ID, m.Path, func(meta *store.Meta) {
		meta.LastUsed = now
	})
	m.LastUsed = now
}

// SelectPreset applies a named preset to the shared Config.
func (a *App) SelectPreset(name string) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	p := a.findPreset(a.alias(), name)
	if p == nil {
		return
	}
	a.applyPreset(a.presetIndex(name))
	a.dirty = false
	a.warn = ""
	a.rememberSelectionLocked()
	a.Log(fmt.Sprintf("preset: %s/%s", a.alias(), name))
	a.emitStateLocked()
}

// UpdateConfig stores the (already collected) form values in the shared
// Config. Saves/loads/preset switches clear the dirty flag; plain edits set it.
func (a *App) UpdateConfig(cfg schema.Config) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	a.cfg = cfg
	a.dirty = true
	a.warn = "" // config changed: any earlier block no longer applies
	a.emitStateLocked()
}

// SaveConfig writes the current config as the current preset name into
// configs/<alias>.json, upserting by preset name.
func (a *App) SaveConfig() {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	alias := a.alias()
	names := a.presetNames()
	if len(names) == 0 {
		a.Log("save FAILED: no preset selected")
		a.emitStateLocked()
		return
	}
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
		a.emitStateLocked()
		return
	}
	a.presets[alias] = kept
	a.dirty = false
	a.Log(fmt.Sprintf("saved configs/%s.json [preset %s]", alias, name))
	a.emitStateLocked()
}

// SavePresetAs saves the current form as a new named preset for the selected
// model, makes it the applied preset and persists configs/<alias>.json.
func (a *App) SavePresetAs(name string) error {
	name = strings.TrimSpace(name)
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	alias := a.alias()
	if alias == "" {
		return fmt.Errorf("select a model first")
	}
	if name == "" {
		return fmt.Errorf("preset name required")
	}
	if a.findPreset(alias, name) != nil {
		return fmt.Errorf("preset %q already exists for %s", name, alias)
	}
	if a.cfg.Basic.Model == "" {
		return fmt.Errorf("select a model first")
	}
	cfg := a.cfg
	cfg.Basic.Alias = alias
	list := append(a.presets[alias], cfg.AsPreset(name))
	if err := store.SaveAlias(settings.ConfigsDir(), alias, list); err != nil {
		a.Log(fmt.Sprintf("save FAILED configs/%s.json: %v", alias, err))
		return err
	}
	a.presets[alias] = list
	a.cfg = cfg
	a.selPreset = len(list) - 1
	a.dirty = false
	a.Log(fmt.Sprintf("saved configs/%s.json [new preset %s]", alias, name))
	a.emitStateLocked()
	return nil
}

// LoadConfig reads configs/<alias>.json from disk and replaces the alias's
// in-memory presets. Missing/malformed file: logged, nothing changed.
func (a *App) LoadConfig() {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	alias := a.alias()
	f, err := store.LoadAlias(settings.ConfigsDir(), alias)
	if err != nil {
		a.Log(fmt.Sprintf("loaded configs/%s.json FAILED: %v (nothing changed)", alias, err))
		a.emitStateLocked()
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
	a.emitStateLocked()
}

// StartServer launches the real llama-server. Errors (port busy,
// missing binary) are logged and leave the state machine untouched. Before
// spawning anything it runs the VRAM pre-flight: a launch that would exceed
// 90% of the GPU's memory is refused outright (that runaway allocation is what
// previously OOM-killed the whole WSL distro).
func (a *App) StartServer() {
	a.stateMu.RLock()
	cfg := a.cfg
	cmdline := a.commandLineLocked()
	presetName := ""
	if names := a.presetNames(); a.selPreset >= 0 && a.selPreset < len(names) {
		presetName = names[a.selPreset]
	}
	a.stateMu.RUnlock()

	est := control.EstimateVram(cfg)
	if est.Warn != "" {
		a.stateMu.Lock()
		a.warn = est.Warn
		a.stateMu.Unlock()
		a.Log("start BLOCKED — " + est.Warn)
		a.Log("start BLOCKED — " + est.Detail)
		a.emitState()
		return
	}
	a.stateMu.Lock()
	a.warn = ""
	a.stateMu.Unlock()
	a.Log(est.Detail)
	a.Log("starting llama-server: " + cmdline)
	// Manager callbacks call emitState, so never hold stateMu across Start.
	if err := a.mgr.Start(cfg); err != nil {
		a.Log(fmt.Sprintf("start FAILED: %v", err))
		a.emitState()
		return
	}
	a.rememberRun(cfg.Basic.Alias, presetName)
	// Freeze the launched config: the Status panel reports THIS, even if the
	// user later picks another model/preset in the form.
	a.stateMu.Lock()
	rc := cfg
	a.runCfg = &rc
	a.runPreset = presetName
	a.metricHistory = nil
	a.metricsAt = time.Time{}
	a.stateMu.Unlock()
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

func (a *App) commandLineLocked() string {
	return "llama-server " + strings.Join(control.BuildArgs(a.cfg), " ")
}

// snapshot returns a deep-enough copy of the current state. The returned
// slices do not alias App state, so callers may marshal them after the lock is
// released.
func (a *App) snapshot() Snapshot {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.snapshotLocked()
}

// snapshotLocked builds a snapshot; the caller must hold stateMu (read or
// write).
func (a *App) snapshotLocked() Snapshot {
	info := a.mgr.Info()
	// Models/categories/boot lines are copied so the JSON encoder never touches
	// a slice that another goroutine may append to.
	models := append([]schema.Model(nil), a.models...)
	for i := range models {
		models[i].Category = append([]string(nil), models[i].Category...)
	}
	cats := make([][]string, len(a.cats))
	for i := range a.cats {
		cats[i] = append([]string(nil), a.cats[i]...)
	}
	boot := append([]string(nil), a.bootLines...)

	// The status panel describes the LOADED server; when nothing is loaded the
	// run config is ignored without mutating it (a later emit may still show a
	// transition).
	runCfg := a.runCfg
	if info.Status == control.StatusStopped || info.Status == control.StatusCrashed {
		runCfg = nil
	}
	runFingerprint := ""
	if runCfg != nil {
		runFingerprint = store.ConfigFingerprint(*runCfg)
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
	if runCfg != nil {
		if runCfg.Basic.Model != "" {
			srv.Model = filepath.Base(runCfg.Basic.Model)
		}
		srv.Context = fmt.Sprint(runCfg.Basic.Context)
		srv.GPULayers = runCfg.Basic.GPULayers
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
	issues, est := control.CheckConfig(a.cfg)
	metrics := append([]schema.MetricsPoint(nil), a.metricHistory...)
	s := Snapshot{
		Status:            string(info.Status),
		CrashExit:         info.Exit,
		Server:            srv,
		Models:            models,
		Categories:        cats,
		SelectedModel:     a.selModel,
		Alias:             a.alias(),
		PresetNames:       a.presetNames(),
		SelectedPreset:    a.selPreset,
		Config:            a.cfg,
		ConfigPath:        fmt.Sprintf("configs/%s.json", a.alias()),
		Dirty:             a.dirty,
		CommandLine:       a.commandLineLocked(),
		BootLines:         boot,
		Warning:           a.warn,
		VramEstimate:      est.Detail,
		VramPercent:       est.GPUPercent,
		VramFits:          est.Fits,
		Validation:        issues,
		Metrics:           metrics,
		Benchmarks:        append([]schema.Benchmark(nil), a.bench...),
		Benchmarking:      a.benchmarking,
		ConfigFingerprint: store.ConfigFingerprint(a.cfg),
		RunFingerprint:    runFingerprint,
		RecentLog:         append([]string(nil), info.RecentLog...),
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
	a.stateMu.Lock()
	a.recordMetricsLocked()
	s := a.snapshotLocked()
	a.stateMu.Unlock()
	a.notifyServerTransition(s.Status, s.CrashExit)
	a.dispatchState(s)
}

// emitStateLocked is emitState for callers that already hold stateMu.
func (a *App) emitStateLocked() {
	a.recordMetricsLocked()
	s := a.snapshotLocked()
	a.notifyServerTransition(s.Status, s.CrashExit)
	a.dispatchState(s)
}

// metricsMax is 10 minutes at the 5 s sampling cadence.
const metricsMax = 120

// recordMetricsLocked appends a sample at most every 5 s while the managed
// server runs. The UI draws it as a sparkline; nothing is persisted.
func (a *App) recordMetricsLocked() {
	now := time.Now()
	if !a.metricsAt.IsZero() && now.Sub(a.metricsAt) < 5*time.Second {
		return
	}
	info := a.mgr.Info()
	if info.Status != control.StatusRunning {
		return
	}
	sp := a.mgr.Speed()
	util, temp := control.GPUUtilTemp()
	a.metricHistory = append(a.metricHistory, schema.MetricsPoint{
		At: now.Unix(), VRAMMB: info.VRAMMB, RAMMB: info.RAMMB,
		TokPerSec: sp.Avg(), GPUUtil: util, GPUTemp: temp,
	})
	if len(a.metricHistory) > metricsMax {
		a.metricHistory = a.metricHistory[len(a.metricHistory)-metricsMax:]
	}
	a.metricsAt = now
}

// dispatchState sends a snapshot to the frontend (or stdout in daemon mode).
func (a *App) dispatchState(s Snapshot) {
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
	a.recordLog(line)
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
	a.initCatalog()
	a.stateMu.Lock()
	a.reloadBenchmarksLocked()
	a.stateMu.Unlock()
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
	a.restoreLastRun()
	a.mgr.AttachExternal(a.cfg.Basic.Host, a.cfg.Basic.Port)
	if a.mgr.Info().Status == control.StatusRunning {
		rc := a.cfg
		a.runCfg = &rc
		if names := a.presetNames(); a.selPreset >= 0 && a.selPreset < len(names) {
			a.runPreset = names[a.selPreset]
		}
	}
	a.refreshDoctor()
	a.ensureAPIToken()
	a.emitState()
}

// ensureAPIToken turns authentication on for an existing install (new installs
// get one when SetPaths writes the config). A literal apiToken "off" opts out.
func (a *App) ensureAPIToken() {
	gen, err := settings.EnsureAPIToken()
	if err != nil {
		a.Log("ctl api: could not generate a token: " + err.Error())
		return
	}
	if gen {
		a.Log("ctl api: generated an access token (stored in the user config)")
	}
}

// initCatalog activates the cached model catalog and refreshes it from the
// repository in the background (never blocks startup, never fails it).
func (a *App) initCatalog() {
	hf.LoadCachedCatalog()
	go a.RefreshCatalog()
}
