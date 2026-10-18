// Package hf downloads GGUF models from the Hugging Face Hub and recommends
// models that fit this machine's hardware. It talks to the public Hub API
// (api/models/<repo>/tree) and the resolve endpoint directly: no Python,
// huggingface-cli or `hf` binary is required, so it works identically in the
// GUI, the CLI and the headless daemon.
//
// Downloads are resumable (HTTP Range), verified against the Hub's size and
// sha256 (LFS oid), and never leave a partial file behind on failure.
package hf

// Entry is one curated, downloadable model. File is either an exact Hub path
// or a case-insensitive suffix (e.g. "Q4_K_M.gguf") resolved against the
// repository's file list at download time, so the catalog survives quant
// renames. MMProj, when set, is the matching vision projector file (same repo).
// Vision/MTP flag capabilities up front; a pasted repo outside the catalog has
// them auto-detected from the downloaded file (GGUF header + sidecar names).
type Entry struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Repo       string  `json:"repo"`
	File       string  `json:"file"`
	MMProj     string  `json:"mmproj,omitempty"`
	ParamsB    float64 `json:"paramsB"`
	Quant      string  `json:"quant"`
	Bytes      int64   `json:"bytes"`      // approximate total download size
	MinVRAMGiB float64 `json:"minVramGiB"` // recommended VRAM with ~50k context
	MinRAMGiB  float64 `json:"minRamGiB"`  // RAM when running on CPU
	Vision     bool    `json:"vision"`
	MTP        bool    `json:"mtp,omitempty"` // ships an embedded MTP (NextN) head
	Context    int     `json:"context"`       // recommended context window
	Note       string  `json:"note"`
	Starter    bool    `json:"starter"` // the small default offered on first run
}

// defaultContext is the ~50k context every curated entry targets.
const defaultContext = 51200

// Catalog is the curated list, ordered smallest to largest. All entries are
// public, permissively licensed GGUF repos.
var Catalog = []Entry{
	{
		ID: "qwen2.5-0.5b-instruct-q4_k_m", Name: "Qwen2.5 0.5B Instruct (Q4_K_M)",
		Repo: "Qwen/Qwen2.5-0.5B-Instruct-GGUF", File: "qwen2.5-0.5b-instruct-q4_k_m.gguf",
		ParamsB: 0.5, Quant: "Q4_K_M", Bytes: 491400032,
		MinVRAMGiB: 1.0, MinRAMGiB: 2.0, Context: defaultContext, Starter: true,
		Note: "Apache-2.0; tiny starter model.",
	},
	{
		ID: "qwen2.5-1.5b-instruct-q4_k_m", Name: "Qwen2.5 1.5B Instruct (Q4_K_M)",
		Repo: "Qwen/Qwen2.5-1.5B-Instruct-GGUF", File: "qwen2.5-1.5b-instruct-q4_k_m.gguf",
		ParamsB: 1.5, Quant: "Q4_K_M", Bytes: 1117320736,
		MinVRAMGiB: 2.0, MinRAMGiB: 3.5, Context: defaultContext,
		Note: "Apache-2.0; fast general chat.",
	},
	{
		ID: "llama-3.2-3b-instruct-q4_k_m", Name: "Llama 3.2 3B Instruct (Q4_K_M)",
		Repo: "bartowski/Llama-3.2-3B-Instruct-GGUF", File: "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
		ParamsB: 3, Quant: "Q4_K_M", Bytes: 2019377696,
		MinVRAMGiB: 3.5, MinRAMGiB: 5.0, Context: defaultContext,
		Note: "Llama 3.2 Community License.",
	},
	{
		ID: "gemma-3-4b-it-q4_k_m", Name: "Gemma 3 4B IT (Q4_K_M, vision)",
		Repo: "ggml-org/gemma-3-4b-it-GGUF", File: "gemma-3-4b-it-Q4_K_M.gguf",
		MMProj:  "mmproj-model-f16.gguf",
		ParamsB: 4, Quant: "Q4_K_M", Bytes: 2489757856 + 851251104,
		MinVRAMGiB: 5.5, MinRAMGiB: 6.5, Vision: true, Context: defaultContext,
		Note: "Gemma Terms of Use; image + text.",
	},
	{
		ID: "qwen2.5-7b-instruct-q4_k_m", Name: "Qwen2.5 7B Instruct (Q4_K_M)",
		Repo: "bartowski/Qwen2.5-7B-Instruct-GGUF", File: "Qwen2.5-7B-Instruct-Q4_K_M.gguf",
		ParamsB: 7, Quant: "Q4_K_M", Bytes: 4683074240,
		MinVRAMGiB: 8.0, MinRAMGiB: 9.0, Context: defaultContext,
		Note: "Apache-2.0; strong text quality.",
	},
	{
		ID: "qwen3-8b-q4_k_m", Name: "Qwen3 8B (Q4_K_M)",
		Repo: "Qwen/Qwen3-8B-GGUF", File: "Qwen3-8B-Q4_K_M.gguf",
		ParamsB: 8, Quant: "Q4_K_M", Bytes: 5027783488,
		MinVRAMGiB: 8.5, MinRAMGiB: 10.0, Context: defaultContext,
		Note: "Apache-2.0; reasoning-capable.",
	},
	{
		ID: "qwen2.5-vl-7b-instruct-q4_k_m", Name: "Qwen2.5-VL 7B Instruct (Q4_K_M, vision)",
		Repo: "unsloth/Qwen2.5-VL-7B-Instruct-GGUF", File: "Qwen2.5-VL-7B-Instruct-Q4_K_M.gguf",
		MMProj:  "mmproj-F16.gguf",
		ParamsB: 7, Quant: "Q4_K_M", Bytes: 4683072384 + 1354163040,
		MinVRAMGiB: 10.5, MinRAMGiB: 11.0, Vision: true, Context: defaultContext,
		Note: "Apache-2.0; image understanding.",
	},
}

// Starter returns the small default model offered during first run.
func Starter() Entry {
	for _, e := range Catalog {
		if e.Starter {
			return e
		}
	}
	return Catalog[0]
}

// ByID returns the catalog entry with the given id.
func ByID(id string) (Entry, bool) {
	for _, e := range Catalog {
		if e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}
