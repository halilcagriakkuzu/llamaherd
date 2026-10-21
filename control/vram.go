// vram.go — pre-flight memory estimate for a llama-server launch. It is a
// guard rail, not a promise: we size the model weights (file size × GPU
// fraction), the KV cache (only the full-attention layers, honouring K/V
// cache quantisation), a small SSM/recurrent term for hybrid models, and a
// compute-buffer allowance. If the total exceeds 90% of the GPU's total VRAM
// the caller refuses to start — that is the failure mode we saw kill the
// whole WSL distro (a runaway allocation spilling to host RAM + swap).
package control

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"llamaherd/schema"
	"llamaherd/store"
)

// BytesPerElement returns the storage cost of one KV element for a llama.cpp
// cache-type name (--cache-type-k/v). ok=false means "unknown, assume f16".
func BytesPerElement(cacheType string) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(cacheType)) {
	case "f32", "fp32":
		return 4.0, true
	case "f16", "fp16", "bf16":
		return 2.0, true
	case "q8_0":
		return 34.0 / 32.0, true // 1.0625
	case "q6_k":
		return 210.0 / 256.0, true // 0.8203
	case "q5_1":
		return 24.0 / 32.0, true // 0.75
	case "q5_0", "q5_k":
		return 176.0 / 256.0, true // 0.6875
	case "q4_1":
		return 20.0 / 32.0, true // 0.625
	case "q4_0", "q4_k", "iq4_nl":
		return 18.0 / 32.0, true // 0.5625
	}
	return 0, false
}

const mib = 1024.0 * 1024.0

// vramBudgetFraction is the share of total GPU memory a launch may predict.
// Above this the pre-flight refuses to start (the requested 90% rule).
const vramBudgetFraction = 0.90

// VramEstimate is the breakdown of a predicted launch footprint (all MiB,
// float so small terms survive rounding).
type VramEstimate struct {
	ModelMB    float64 // weights placed on the GPU
	KVKMB      float64
	KVVMB      float64
	SSMMB      float64 // recurrent/conv state of hybrid layers
	ComputeMB  float64 // graph/compute buffer + CUDA context
	MMProjMB   float64 // vision sidecar on the GPU (unless --no-mmproj-offload)
	DraftMB    float64 // separate draft/sidecar model placed on the GPU
	TotalMB    float64
	TotalVram  int     // GPU total VRAM (0 = unknown)
	BudgetMB   float64 // 90% of TotalVram
	GPUPercent float64 // TotalMB / TotalVram * 100
	Fits       bool
	Detail     string
	Warn       string // non-empty => do not start
}

// EstimateVram predicts the GPU footprint of cfg. path resolution and GGUF
// parsing are best-effort: if metadata cannot be read, only the weight cost
// (file size × GPU fraction) is reported and the result is conservative.
func EstimateVram(cfg schema.Config) VramEstimate {
	var e VramEstimate

	gpuFrac, layerNote := gpuLayerFraction(cfg)

	if cfg.Basic.Model == "" {
		e.Detail = "no model selected"
		return e
	}
	st, err := os.Stat(cfg.Basic.Model)
	if err != nil {
		e.Detail = "model file could not be read: " + err.Error()
		return e
	}
	modelBytes := float64(st.Size())
	e.ModelMB = modelBytes / mib * gpuFrac
	if info0, err := store.ReadGGUFInfo(cfg.Basic.Model); err == nil {
		e.ModelMB -= overrideCPUMB(cfg.Advanced.OverrideTensor, info0.BlockCount, modelBytes/mib)
		if e.ModelMB < 0 {
			e.ModelMB = 0
		}
	}

	info, metaErr := store.ReadGGUFInfo(cfg.Basic.Model)
	weightsKnown := metaErr == nil && info.KeyLength > 0 && info.FullAttentionLayers() > 0

	ctx := cfg.Basic.Context
	if ctx <= 0 {
		ctx = 4096
	}
	if weightsKnown {
		fa := info.FullAttentionLayers()
		kvHeads := info.HeadCountKV
		if kvHeads <= 0 {
			kvHeads = info.HeadCount
		}
		bpeK, okK := BytesPerElement(cfg.Basic.KCache)
		if !okK {
			bpeK = 2.0
		}
		bpeV, okV := BytesPerElement(cfg.Basic.VCache)
		if !okV {
			bpeV = 2.0
		}
		e.KVKMB = float64(ctx) * float64(fa) * float64(kvHeads) * float64(info.KeyLength) * bpeK / mib
		e.KVVMB = float64(ctx) * float64(fa) * float64(kvHeads) * float64(info.ValueLength) * bpeV / mib
		// Recurrent (SSM) layers hold a context-independent state per layer.
		if info.FullAttentionInterval > 1 {
			nSSM := info.BlockCount - fa
			if nSSM > 0 && info.SSMInnerSize > 0 {
				elems := info.SSMConvKernel*info.SSMInnerSize + info.SSMStateSize*info.SSMInnerSize
				e.SSMMB = float64(nSSM) * float64(elems) * 2 / mib
			}
			if e.SSMMB < 32 {
				e.SSMMB = 32 // floor so the term is never misleadingly zero
			}
		}
	}

	// Compute/graph buffer + CUDA context. The graph scales with the batch we
	// feed it, so a flat base plus a small context term is a reasonable
	// stand-in (llama.cpp's compute buffer is not proportional to n_ctx).
	base := 512.0
	if ub := cfg.Advanced.UBatch; ub > 0 && ub < 512 {
		base = 300 + float64(ub)*0.4 // ~512 at ub=512, smaller below
	}
	e.ComputeMB = base + float64(ctx)/1024.0*2 + e.ModelMB*0.01

	// Vision sidecar stays on the GPU unless offloaded away.
	if cfg.Advanced.MMProj != "" && !cfg.Advanced.NoMMProjOffload {
		if mmst, err := os.Stat(cfg.Advanced.MMProj); err == nil {
			e.MMProjMB = float64(mmst.Size()) / mib
		}
	}

	// A separate draft model (--spec-draft-model) shares the GPU with the
	// target; its weights and KV are real VRAM the guard must reserve.
	if cfg.Advanced.DraftModel != "" {
		if dst, err := os.Stat(cfg.Advanced.DraftModel); err == nil {
			e.DraftMB = float64(dst.Size()) / mib
		}
	}

	e.TotalMB = e.ModelMB + e.KVKMB + e.KVVMB + e.SSMMB + e.ComputeMB + e.MMProjMB + e.DraftMB

	e.TotalVram = TotalVramMB()
	if e.TotalVram > 0 {
		e.BudgetMB = float64(e.TotalVram) * vramBudgetFraction
		e.GPUPercent = e.TotalMB / float64(e.TotalVram) * 100
		e.Fits = e.TotalMB <= e.BudgetMB
	} else {
		e.Fits = true // no GPU visible: nothing to guard against
	}

	attn := ""
	if weightsKnown && info.FullAttentionInterval > 1 {
		attn = fmt.Sprintf(", %d/%d layers full-attention", info.FullAttentionLayers(), info.BlockCount)
	}
	meta := layerNote + attn
	if metaErr != nil {
		meta += ", architecture could not be read (rough estimate)"
	}
	e.Detail = fmt.Sprintf("estimated VRAM %.1f GiB / %.1f GiB (%.0f%%) [model %.1f, KV %.2f+%.2f, SSM %.2f, compute %.2f; %s]",
		e.TotalMB/1024, float64(e.TotalVram)/1024, e.GPUPercent,
		e.ModelMB/1024, e.KVKMB/1024, e.KVVMB/1024, e.SSMMB/1024, e.ComputeMB/1024, meta)

	if !e.Fits {
		e.Warn = fmt.Sprintf("start blocked: estimated VRAM %.1f GiB, card's 90%% budget %.1f GiB (total %.1f GiB). Reduce the context and/or the q4 KV cache, or lower the GPU layer count (-ngl).",
			e.TotalMB/1024, e.BudgetMB/1024, float64(e.TotalVram)/1024)
	}
	return e
}

// gpuLayerFraction maps the -ngl field ("all", a number, empty) to the share
// of weights expected on the GPU, plus a short note for the detail line.
func gpuLayerFraction(cfg schema.Config) (float64, string) {
	gl := strings.TrimSpace(strings.ToLower(cfg.Basic.GPULayers))
	if gl == "" || gl == "all" || gl == "auto" {
		return 1.0, "all layers on GPU"
	}
	n, err := strconv.Atoi(gl)
	if err != nil || n < 0 {
		return 1.0, "-ngl=" + gl
	}
	layers := 0
	if info, err := store.ReadGGUFInfo(cfg.Basic.Model); err == nil {
		layers = info.BlockCount
	}
	if layers <= 0 {
		return 1.0, fmt.Sprintf("-ngl=%d", n)
	}
	frac := float64(n) / float64(layers)
	if frac > 1 {
		frac = 1
	}
	return frac, fmt.Sprintf("-ngl=%d/%d", n, layers)
}

// SafeContext returns the largest context (in 1024-token steps) that still
// passes the VRAM guard for cfg with its current -ngl. It never exceeds the
// model's trained context or desired, and returns desired unchanged when no GPU
// is visible (there is nothing to guard).
func SafeContext(cfg schema.Config, desired int) int {
	if desired <= 0 {
		desired = 32768
	}
	if info, err := store.ReadGGUFInfo(cfg.Basic.Model); err == nil && info.ContextLength > 0 && desired > info.ContextLength {
		desired = info.ContextLength
	}
	if TotalVramMB() <= 0 {
		return desired
	}
	for ctx := desired; ctx >= 2048; ctx -= 1024 {
		c := cfg
		c.Basic.Context = ctx
		if EstimateVram(c).Fits {
			return ctx
		}
	}
	return 2048
}

// FitPreset adjusts a preset to the current machine: pick the largest context
// that fits with all layers on the GPU, then use the largest -ngl that still
// fits at that context. This replaces hardcoded "51200 context + all layers"
// presets that force a tiny GPU offload on smaller cards.
func FitPreset(cfg schema.Config, desiredContext int) schema.Config {
	c := cfg
	c.Basic.Context = SafeContext(c, desiredContext)
	if ngl := SafeGpuLayers(c); ngl >= 0 {
		c.Basic.GPULayers = strconv.Itoa(ngl)
	}
	return c
}

// SafeGpuLayers returns the largest -ngl (0..layers) whose launch estimate
// stays within the budget, or -1 when it cannot be determined (no GPU, or the
// model metadata is unreadable). A value equal to the model's layer count
// means "everything fits on the GPU".
func SafeGpuLayers(cfg schema.Config) int {
	if TotalVramMB() <= 0 {
		return -1
	}
	info, err := store.ReadGGUFInfo(cfg.Basic.Model)
	if err != nil || info.BlockCount <= 0 {
		return -1
	}
	for ngl := info.BlockCount; ngl >= 0; ngl-- {
		c := cfg
		c.Basic.GPULayers = strconv.Itoa(ngl)
		if EstimateVram(c).Fits {
			return ngl
		}
	}
	return -1
}

// vramProbe, when non-nil, replaces the provider query (tests only).
var vramProbe func() int

// TotalVramMB reports the active provider's total GPU memory in MiB (0 when no
// provider is available). Cached by ActiveGPU so a UI keystroke storm cannot
// fork a process per key.
func TotalVramMB() int {
	if f := vramProbe; f != nil {
		return f()
	}
	return ActiveGPU().TotalVRAMMB()
}

// overrideCPUMB estimates the weight MiB moved to the CPU by -ot rules whose
// target is CPU. Each layer's FFN is ~65% of its bytes, split over 3 tensors.
func overrideCPUMB(spec string, layers int, modelMB float64) float64 {
	if spec == "" || layers <= 0 {
		return 0
	}
	perTensor := modelMB * 0.65 / float64(layers) / 3
	var total float64
	for _, rule := range strings.Split(spec, ",") {
		i := strings.LastIndex(rule, "=")
		if i < 0 || !strings.EqualFold(strings.TrimSpace(rule[i+1:]), "CPU") {
			continue
		}
		re, err := regexp.Compile(rule[:i])
		if err != nil {
			continue
		}
		for l := 0; l < layers; l++ {
			for _, t := range []string{"ffn_up", "ffn_gate", "ffn_down"} {
				if re.MatchString(fmt.Sprintf("blk.%d.%s.weight", l, t)) {
					total += perTensor
				}
			}
		}
	}
	return total
}
