// Package schema holds the shared data types: server status rows, the
// basic+advanced launch config, presets and the model list rows.
package schema

// Server is the llama-server state shown by the status panel (task 02).
type Server struct {
	Status    string `json:"status"` // starting | running | stopped | crashed
	PID       string `json:"pid"`
	Binary    string `json:"binary"`
	Listen    string `json:"listen"`
	WSLIP     string `json:"wslIp"`
	Model     string `json:"model"`
	Context   string `json:"context"`
	GPULayers string `json:"gpuLayers"`
	VRAM      string `json:"vram"`
	CPURAM    string `json:"cpuRam"`
	Uptime    string `json:"uptime"`
}

// Basic holds the BASIC tab field values (task 04).
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

// Advanced holds the ADVANCED tab field values (task 05).
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
	SleepIdle       int    `json:"sleepIdle"`
	CacheRAM        int    `json:"cacheRAM"`
	Mlock           bool   `json:"mlock"`
	NoMmap          bool   `json:"noMmap"`
	CPUThreads      int    `json:"cpuThreads"`
	UBatch          int    `json:"ubatch"`
	Batch           int    `json:"batch"`
	OverrideTensor  string `json:"overrideTensor"`
}

// Config is the ONE shared config shape (README "One typed config"): both
// forms, all presets, command-line assembly and phase-2 persistence
// (configs/<alias>.json) read/wrt this struct's basic+advanced parts.
type Config struct {
	Basic    Basic    `json:"basic"`
	Advanced Advanced `json:"advanced"`
}

// Preset is a named full basic+advanced field set for an alias (task 06).
type Preset struct {
	Name     string   `json:"name"`
	Basic    Basic    `json:"basic"`
	Advanced Advanced `json:"advanced"`
}

// ToPreset view helpers -----------------------------------------------

// AsConfig returns the preset as the shared Config shape.
func (p *Preset) AsConfig() Config {
	return Config{Basic: p.Basic, Advanced: p.Advanced}
}

// AsPreset converts a Config into a Preset (used by save in phase 2).
func (c Config) AsPreset(name string) Preset {
	return Preset{Name: name, Basic: c.Basic, Advanced: c.Advanced}
}

// Model is one row of the model list (task 03).
type Model struct {
	Display  string `json:"display"`
	Path     string `json:"path"`
	Alias    string `json:"alias"`
	IsMMProj bool   `json:"isMMProj"`
	// Library metadata (store.Meta): category path (e.g. Qwen > 3.8 >
	// Uncensored), the paired vision sidecar and the model's origin.
	Category []string `json:"category"`
	MMProj   string   `json:"mmproj"`
	MTP      bool     `json:"mtp"`
	// MTPDraft is a separate MTP draft sidecar (--spec-draft-model) recorded for
	// this model, empty when none is installed.
	MTPDraft string `json:"mtpDraft,omitempty"`
	Source   string `json:"source"`
}
