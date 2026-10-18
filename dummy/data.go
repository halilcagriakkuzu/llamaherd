package dummy

// Server is the (dummy) llama-server state shown by the status panel (task 02).
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

var ServerState = Server{
	Status:    "running",
	PID:       "749941",
	Binary:    "llama-server", // overridden at runtime from settings.ServerBin()
	Listen:    "127.0.0.1:8080",
	WSLIP:     "", // filled at runtime only inside WSL (see app.snapshot)
	Model:     "Qwen3.8-27B-UD-IQ4_XS.gguf",
	Context:   "25556",
	GPULayers: "33/33",
	VRAM:      "16.8 GB",
	CPURAM:    "1.2 GB",
	Uptime:    "01:23:45",
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

// Models are the four list rows in display order (README "Models" table).
var Models = []Model{
	{
		Display: "Qwen3.8-27B-UD-IQ4_XS",
		Path:    "unsloth/Qwen3.8-27B-GGUF/Qwen3.8-27B-UD-IQ4_XS.gguf",
		Alias:   "qwen3.8-27b-local",
	},
	{
		Display: "Qwen3.8-27B-TurboFCFusion-...-IQ4_XS",
		Path:    "Qwen3.8_Uncensored/Heretic/Fable-Distilled/Qwen3.8-27B-TurboFCFusion-735-882-Here-Uncen-NEO-CODER-MAX-LOW-MTP-IQ4_XS.gguf",
		Alias:   "qwen3.8-27b-uncensored-local",
	},
	{
		Display: "Qwen3.8-27B-Uncensored-HauhauCS-Q3_K_P",
		Path:    "Qwen3.8-27B-Uncensored-HauhauCS-Aggressive-Q3_K_P.gguf",
		Alias:   "qwen3.8-27b-uncensored-hauhaucs",
	},
	{
		Display:  "mmproj-F16.gguf",
		Path:     "Qwen3.8_Uncensored/Heretic/Fable-Distilled/mmproj-F16.gguf",
		IsMMProj: true,
	},
}

// The TurboFCFusion model path (model 2) reused by the vision+mtp preset.
const turboFCFusionPath = "Qwen3.8_Uncensored/Heretic/Fable-Distilled/Qwen3.8-27B-TurboFCFusion-735-882-Here-Uncen-NEO-CODER-MAX-LOW-MTP-IQ4_XS.gguf"

const mmprojPath = "Qwen3.8_Uncensored/Heretic/Fable-Distilled/mmproj-F16.gguf"

// Presets maps alias -> named presets (README "Presets" table). The three real
// reference commands become three presets; the HauhauCS preset is derived.
var Presets = map[string][]Preset{
	"qwen3.8-27b-local": {
		{
			Name: "default",
			Basic: Basic{
				Model:   "unsloth/Qwen3.8-27B-GGUF/Qwen3.8-27B-UD-IQ4_XS.gguf",
				Alias:   "qwen3.8-27b-local",
				Host:    "127.0.0.1",
				Port:    8090,
				Context: 8192,
				// 60/65 layers on GPU keeps the launch under 90% of the 16 GB
				// card (weights 13.3 GiB alone do not fit with q8 KV).
				GPULayers:       "60",
				ParallelSlots:   1,
				FlashAttention:  "on",
				KCache:          "q8_0",
				VCache:          "q8_0",
				Jinja:           true,
				ReasoningFmt:    "deepseek",
				ReasoningBudget: 8096,
			},
			Advanced: Advanced{
				SpecType:  "draft-mtp",
				DraftNMax: 3,
				Fit:       "off",
				Perf:      true,
				SleepIdle: 300,
			},
		},
	},
	"qwen3.8-27b-uncensored-local": {
		{
			Name: "vision+mtp",
			Basic: Basic{
				Model:   turboFCFusionPath,
				Alias:   "qwen3.8-27b-uncensored-local",
				Host:    "127.0.0.1",
				Port:    8090,
				Context: 32768,
				// Vision adds the mmproj sidecar on the GPU, so fewer layers
				// fit: 51/65 stays under 90% of the 16 GB card.
				GPULayers:       "51",
				ParallelSlots:   1,
				FlashAttention:  "on",
				KCache:          "q8_0",
				VCache:          "q8_0",
				Jinja:           true,
				ReasoningFmt:    "deepseek",
				ReasoningBudget: 8096,
			},
			Advanced: Advanced{
				SpecType:        "draft-mtp",
				DraftNMax:       2,
				DraftNgl:        "all",
				DraftTypeK:      "q8_0",
				DraftTypeV:      "q8_0",
				MMProj:          mmprojPath,
				NoMMProjOffload: true,
				Fit:             "off",
				Perf:            false,
				SleepIdle:       5,
				CacheRAM:        4096,
			},
		},
		{
			Name: "plain",
			Basic: Basic{
				Model:           turboFCFusionPath,
				Alias:           "qwen3.8-27b-uncensored-local",
				Host:            "127.0.0.1",
				Port:            8090,
				Context:         32768,
				GPULayers:       "55",
				ParallelSlots:   1,
				FlashAttention:  "on",
				KCache:          "q8_0",
				VCache:          "q8_0",
				Jinja:           true,
				ReasoningFmt:    "deepseek",
				ReasoningBudget: 8096,
			},
			Advanced: Advanced{
				SpecType:  "none",
				Fit:       "off",
				Perf:      false,
				SleepIdle: 5,
			},
		},
	},
	"qwen3.8-27b-uncensored-hauhaucs": {
		{
			Name: "default",
			Basic: Basic{
				Model:           "Qwen3.8-27B-Uncensored-HauhauCS-Aggressive-Q3_K_P.gguf",
				Alias:           "qwen3.8-27b-uncensored-hauhaucs",
				Host:            "127.0.0.1",
				Port:            8090,
				Context:         32768,
				GPULayers:       "62",
				ParallelSlots:   1,
				FlashAttention:  "on",
				KCache:          "q8_0",
				VCache:          "q8_0",
				Jinja:           true,
				ReasoningFmt:    "deepseek",
				ReasoningBudget: 8096,
			},
			Advanced: Advanced{
				SpecType:  "none",
				Fit:       "off",
				Perf:      false,
				SleepIdle: 300,
			},
		},
	},
}

// PresetNames returns the preset names for an alias, in order.
func PresetNames(alias string) []string {
	var names []string
	for _, p := range Presets[alias] {
		names = append(names, p.Name)
	}
	return names
}

// FindPreset returns the named preset for an alias, or nil if not found.
func FindPreset(alias, name string) *Preset {
	for i := range Presets[alias] {
		if Presets[alias][i].Name == name {
			return &Presets[alias][i]
		}
	}
	return nil
}
