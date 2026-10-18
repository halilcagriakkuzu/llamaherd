package hf

import (
	"fmt"
	"sort"
)

// Fit values, best first.
const (
	FitGPU    = "gpu"    // fits entirely in VRAM
	FitHybrid = "hybrid" // at least half of the weights fit in VRAM, the rest runs from RAM
	FitCPU    = "cpu"    // small enough to run from system RAM
	FitNone   = "none"   // does not suit this machine (only in the full catalog view)
)

const (
	gib = float64(1 << 30)
	// guard mirrors the 90% VRAM budget enforced before a start (control/vram.go).
	guard = 0.90
	// minHybridShare is the smallest fraction of weights on the GPU that still
	// gives usable speed; below it a model is not recommended.
	minHybridShare = 0.5
	// cpuDenseMaxGiB / cpuMoEMaxGiB cap what is suggested for system-RAM-only use:
	// bigger dense models crawl, mixture-of-experts models (few active parameters)
	// stay usable up to a larger size.
	cpuDenseMaxGiB = 6.5
	cpuMoEMaxGiB   = 12.0
)

// Rec is one catalog entry scored against a machine's hardware.
type Rec struct {
	Entry Entry   `json:"entry"`
	Fit   string  `json:"fit"`
	Share float64 `json:"share,omitempty"` // fraction of weights on the GPU (gpu/hybrid)
	Why   string  `json:"why"`
	// MaxContext is the largest context (tokens) this card can hold with the
	// model fully on the GPU and a q8_0 KV cache; 0 = unknown.
	MaxContext int `json:"maxContext,omitempty"`
}

// fitFor classifies one entry; ok=false means it does not suit this machine and
// must not be shown.
func fitFor(e Entry, vramMB, ramMB int) (Rec, bool) {
	vramGiB := float64(vramMB) / 1024
	ramGiB := float64(ramMB) / 1024
	weights := float64(e.Bytes) / gib
	r := Rec{Entry: e}

	if vramMB > 0 && e.MinVRAMGiB <= vramGiB {
		r.Fit, r.Share = FitGPU, 1
		r.MaxContext = maxContext(e, vramGiB)
		r.Why = fmt.Sprintf("fits %.1f GiB VRAM with ~%dk context", vramGiB, e.Context/1000)
		if r.MaxContext >= 2*e.Context {
			r.Why = fmt.Sprintf("fits %.1f GiB VRAM; up to ~%dk context", vramGiB, r.MaxContext/1000)
		}
		return r, true
	}
	if vramMB > 0 {
		// KV cache + compute buffers stay on the GPU; weights spill into RAM.
		overhead := e.MinVRAMGiB*guard - weights
		if overhead < 0.5 {
			overhead = 0.5
		}
		share := (vramGiB*guard - overhead) / weights
		if share >= minHybridShare && ramGiB >= (1-share)*weights+2 {
			r.Fit, r.Share = FitHybrid, share
			r.Why = fmt.Sprintf("~%.0f%% of the weights fit in %.1f GiB VRAM, the rest runs from RAM (slower)", share*100, vramGiB)
			return r, true
		}
	}
	limit := cpuDenseMaxGiB
	if e.MoE {
		limit = cpuMoEMaxGiB
	}
	if ramGiB > 0 && e.MinRAMGiB <= ramGiB && weights <= limit {
		r.Fit = FitCPU
		r.Why = fmt.Sprintf("runs from system RAM (needs ~%.1f GiB)", e.MinRAMGiB)
		return r, true
	}
	return r, false
}

// maxContext estimates the largest context window that still passes the VRAM
// guard with all layers on the GPU: what remains after weights and fixed
// buffers, divided by the KV growth per token, capped at the trained context.
func maxContext(e Entry, vramGiB float64) int {
	if e.KVMiBPer1k <= 0 {
		return 0
	}
	weights := float64(e.Bytes) / gib
	kvAtRef := e.KVMiBPer1k * float64(e.Context) / 1000 / 1024
	fixed := e.MinVRAMGiB*guard - weights - kvAtRef
	if fixed < 0.3 {
		fixed = 0.3
	}
	avail := vramGiB*guard - fixed - weights
	if avail <= 0 {
		return 0
	}
	tokens := int(avail / (e.KVMiBPer1k / 1024) * 1000)
	tokens -= tokens % 1024
	if e.TrainedContext > 0 && tokens > e.TrainedContext {
		tokens = e.TrainedContext
	}
	if tokens < e.Context {
		tokens = e.Context
	}
	return tokens
}

// Recommend returns the catalog models that suit a machine with vramMB of GPU
// memory and ramMB of system RAM. Models that do not fit are left out. Of each
// Family only the best fit is kept (GPU over hybrid over CPU, then the highest
// quality quant). Result order: fit, then model size, then quant size.
func Recommend(vramMB, ramMB int) []Rec {
	rank := map[string]int{FitGPU: 0, FitHybrid: 1, FitCPU: 2}
	best := map[string]Rec{}
	var order []string
	for _, e := range Entries() {
		r, ok := fitFor(e, vramMB, ramMB)
		if !ok {
			continue
		}
		fam := e.Family
		if fam == "" {
			fam = e.ID
		}
		cur, have := best[fam]
		if !have {
			order = append(order, fam)
		}
		if !have || rank[r.Fit] < rank[cur.Fit] ||
			(rank[r.Fit] == rank[cur.Fit] && e.Bytes > cur.Entry.Bytes) {
			best[fam] = r
		}
	}
	recs := make([]Rec, 0, len(order))
	for _, fam := range order {
		recs = append(recs, best[fam])
	}
	sort.SliceStable(recs, func(i, j int) bool {
		a, b := recs[i], recs[j]
		if rank[a.Fit] != rank[b.Fit] {
			return rank[a.Fit] < rank[b.Fit]
		}
		if a.Entry.ParamsB != b.Entry.ParamsB {
			return a.Entry.ParamsB > b.Entry.ParamsB
		}
		return a.Entry.Bytes > b.Entry.Bytes
	})
	return recs
}

// FamilyCount is the number of distinct models in the catalog (quants of one
// family count once).
func FamilyCount() int {
	seen := map[string]bool{}
	for _, e := range Entries() {
		fam := e.Family
		if fam == "" {
			fam = e.ID
		}
		seen[fam] = true
	}
	return len(seen)
}

// Browse returns every catalog entry scored for the machine, grouped by family
// (best-fitting family first, quants largest first). Entries that do not suit the
// machine are included with Fit == FitNone so the full list can be inspected.
func Browse(vramMB, ramMB int) []Rec {
	rank := map[string]int{FitGPU: 0, FitHybrid: 1, FitCPU: 2, FitNone: 3}
	var all []Rec
	for _, e := range Entries() {
		r, ok := fitFor(e, vramMB, ramMB)
		if !ok {
			r.Fit = FitNone
			r.Why = fmt.Sprintf("needs ~%.1f GiB VRAM or ~%.1f GiB RAM", e.MinVRAMGiB, e.MinRAMGiB)
		}
		all = append(all, r)
	}
	famOf := func(e Entry) string {
		if e.Family != "" {
			return e.Family
		}
		return e.ID
	}
	bestRank := map[string]int{}
	famParams := map[string]float64{}
	for _, r := range all {
		f := famOf(r.Entry)
		if cur, ok := bestRank[f]; !ok || rank[r.Fit] < cur {
			bestRank[f] = rank[r.Fit]
		}
		famParams[f] = r.Entry.ParamsB
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		fa, fb := famOf(a.Entry), famOf(b.Entry)
		if fa != fb {
			if bestRank[fa] != bestRank[fb] {
				return bestRank[fa] < bestRank[fb]
			}
			if famParams[fa] != famParams[fb] {
				return famParams[fa] > famParams[fb]
			}
			return fa < fb
		}
		return a.Entry.Bytes > b.Entry.Bytes
	})
	return all
}
