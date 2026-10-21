// Package schema holds the shared data types: server status rows, the
// basic+advanced launch config, presets and the model list rows.
package schema

// Server is the llama-server state shown by the status panel.
type Server struct {
	Status    string `json:"status"` // starting | running | stopped | crashed
	PID       string `json:"pid"`
	Binary    string `json:"binary"`
	Listen    string `json:"listen"`
	WSLIP     string `json:"wslIp"`
	Model     string `json:"model"`
	Context   string `json:"context"`
	GPULayers string `json:"gpuLayers"`
	// TokPerSec is the average generation speed (e.g. "42.3 tok/s"); the tooltip
	// carries min / max / last. "—" until a request has finished.
	TokPerSec    string `json:"tokPerSec"`
	TokPerSecTip string `json:"tokPerSecTip"`
	VRAM         string `json:"vram"`
	CPURAM       string `json:"cpuRam"`
	Uptime       string `json:"uptime"`
}

// Basic holds the BASIC tab field values.
type Basic struct {
	Model           string `json:"model"`
	Alias           string `json:"alias"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Context         int    `json:"context"`
	GPULayers       string `json:"gpuLayers"`
	ParallelSlots   int    `json:"parallelSlots"`
	FlashAttention  string `json:"flashAttention"`
	KCache          string `json:"kCache"`
	VCache          string `json:"vCache"`
	Jinja           bool   `json:"jinja"`
	ReasoningFmt    string `json:"reasoningFmt"`
	ReasoningBudget int    `json:"reasoningBudget"`
}

// Advanced holds the ADVANCED tab field values.
type Advanced struct {
	SpecType   string `json:"specType"`
	DraftNMax  int    `json:"draftNMax"`
	DraftNgl   string `json:"draftNgl"`
	DraftTypeK string `json:"draftTypeK"`
	DraftTypeV string `json:"draftTypeV"`
	// DraftModel is a separate draft/sidecar GGUF for speculative decoding
	// (--spec-draft-model). Used with a draft-* spec type other than draft-mtp
	// (which reads MTP heads from the target model itself).
	DraftModel      string `json:"draftModel"`
	MMProj          string `json:"mmproj"`
	NoMMProjOffload bool   `json:"noMMProjOffload"`
	Fit             string `json:"fit"`
	Perf            bool   `json:"perf"`
	// Metrics enables llama-server's Prometheus endpoint (--metrics); benchmark
	// runs use it to record slots/queue/context counters.
	Metrics        bool   `json:"metrics"`
	SleepIdle      int    `json:"sleepIdle"`
	CacheRAM       int    `json:"cacheRAM"`
	Mlock          bool   `json:"mlock"`
	NoMmap         bool   `json:"noMmap"`
	CPUThreads     int    `json:"cpuThreads"`
	UBatch         int    `json:"ubatch"`
	Batch          int    `json:"batch"`
	OverrideTensor string `json:"overrideTensor"`
}

// Config is the one shared config shape: both forms, all presets, command-line
// assembly and persistence (configs/<alias>.json) use its basic+advanced parts.
type Config struct {
	Basic    Basic    `json:"basic"`
	Advanced Advanced `json:"advanced"`
}

// Preset is a named full basic+advanced field set for an alias.
type Preset struct {
	Name     string   `json:"name"`
	Basic    Basic    `json:"basic"`
	Advanced Advanced `json:"advanced"`
}

// Preset <-> Config conversions ---------------------------------------

// AsConfig returns the preset as the shared Config shape.
func (p *Preset) AsConfig() Config {
	return Config{Basic: p.Basic, Advanced: p.Advanced}
}

// AsPreset converts a Config into a named Preset.
func (c Config) AsPreset(name string) Preset {
	return Preset{Name: name, Basic: c.Basic, Advanced: c.Advanced}
}

// Model is one row of the model list.
type Model struct {
	// ID is the stable library/config identifier. It equals Alias unless two
	// files share a base name, in which case a short path hash disambiguates it.
	ID       string `json:"id"`
	Display  string `json:"display"`
	Path     string `json:"path"`
	Alias    string `json:"alias"`
	IsMMProj bool   `json:"isMMProj"`
	// Favorite pins a model at the top of the list (library metadata).
	Favorite bool `json:"favorite,omitempty"`
	// LastUsed is the Unix time this model was last selected (0 = never).
	LastUsed int64 `json:"lastUsed,omitempty"`
	// GGUF header details (empty/zero when the file could not be parsed).
	Arch           string `json:"arch,omitempty"`
	Layers         int    `json:"layers,omitempty"`
	Heads          int    `json:"heads,omitempty"`
	KVHeads        int    `json:"kvHeads,omitempty"`
	FullAttnLayers int    `json:"fullAttnLayers,omitempty"`
	TrainedContext int    `json:"trainedContext,omitempty"`
	FileSize       int64  `json:"fileSize,omitempty"`
	Quant          string `json:"quant,omitempty"`
	// Library metadata (store.Meta): category path (e.g. Qwen > 3.8 >
	// Uncensored), the paired vision sidecar and the model's origin.
	Category []string `json:"category"`
	MMProj   string   `json:"mmproj"`
	MTP      bool     `json:"mtp"`
	// MTPDraft is a separate MTP draft sidecar (--spec-draft-model) recorded for
	// this model, empty when none is installed.
	MTPDraft string `json:"mtpDraft,omitempty"`
	Source   string `json:"source"`
	// Provenance is set for models downloaded from Hugging Face; nil for local
	// files.
	Provenance *Provenance `json:"provenance,omitempty"`
}

// Provenance is the Hugging Face repository a model was downloaded from and
// the licence it declared at that time (library metadata, informational only).
type Provenance struct {
	Repo string `json:"repo"`
	URL  string `json:"url"`
	// License is the card's licence id; empty when it was not recorded.
	License     string `json:"license,omitempty"`
	LicenseName string `json:"licenseName,omitempty"`
}

// MetricsPoint is one live sample for the performance charts (5 s apart).
type MetricsPoint struct {
	At        int64   `json:"at"`
	VRAMMB    int     `json:"vramMB"`
	RAMMB     int     `json:"ramMB"`
	TokPerSec float64 `json:"tokPerSec"`
	// GPUUtil (%) and GPUTemp (°C) are 0 when nvidia-smi is unavailable.
	GPUUtil int `json:"gpuUtil"`
	GPUTemp int `json:"gpuTemp"`
}

// Issue is one form/launch validation finding. Field matches the form's
// data-key (e.g. "context") so the UI can mark the input.
type Issue struct {
	Field    string `json:"field"`
	Severity string `json:"severity"` // "error" | "warn"
	Message  string `json:"message"`
}

// ImportCheck is the preview result for one file the user wants to import.
type ImportCheck struct {
	Path          string `json:"path"`
	Size          int64  `json:"size"`
	Duplicate     bool   `json:"duplicate"`
	ExistingAlias string `json:"existingAlias,omitempty"`
	ExistingPath  string `json:"existingPath,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ImportItem is one per-file outcome of a batch import.
type ImportItem struct {
	Source  string `json:"source"`
	Dest    string `json:"dest,omitempty"`
	Alias   string `json:"alias,omitempty"`
	Status  string `json:"status"` // imported | duplicate | error
	Message string `json:"message,omitempty"`
}

// ImportReport is the result of ImportModels.
type ImportReport struct {
	Items []ImportItem `json:"items"`
}

// Benchmark is one stored benchmark run: the measured speed of one exact
// model + config combination (the config is identified by Fingerprint).
type Benchmark struct {
	ModelID     string `json:"modelId"`
	Model       string `json:"model"`
	Alias       string `json:"alias"`
	Preset      string `json:"preset"`
	Fingerprint string `json:"fingerprint"`
	// At is the Unix time of the run.
	At         int64   `json:"at"`
	Context    int     `json:"context"`
	GPULayers  string  `json:"gpuLayers"`
	PromptN    int     `json:"promptN"`
	PromptTokS float64 `json:"promptTokPerSec"`
	GenN       int     `json:"genN"`
	GenTokS    float64 `json:"genTokPerSec"`
	PeakVRAMMB int     `json:"peakVramMB"`
	DurationMS int64   `json:"durationMs"`
	// /metrics counters captured around the run (empty/zero on older builds).
	PromptEvalMS  float64 `json:"promptEvalMs,omitempty"`
	SlotsActive   int     `json:"slotsActive,omitempty"`
	SlotsBusy     int     `json:"slotsBusy,omitempty"`
	QueueDepth    int     `json:"queueDepth,omitempty"`
	ContextTokens int     `json:"contextTokens,omitempty"`
	KVCacheRatio  float64 `json:"kvCacheRatio,omitempty"`
}
