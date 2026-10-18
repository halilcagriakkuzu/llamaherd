package hf

import (
	"fmt"
	"sort"
)

// Rec is one catalog entry scored against a machine's hardware.
type Rec struct {
	Entry Entry  `json:"entry"`
	Fit   string `json:"fit"` // "gpu" | "cpu" | "too-big"
	Why   string `json:"why"`
}

// Recommend ranks the catalog for a machine with vramMB of GPU memory and
// ramMB of system RAM. Models that fit the GPU come first (largest that fits),
// then CPU-only fits, then the rest. Vision entries already carry the projector
// size in MinVRAMGiB, so nothing special is needed to rank them.
func Recommend(vramMB, ramMB int) []Rec {
	vramGiB := float64(vramMB) / 1024
	ramGiB := float64(ramMB) / 1024
	recs := make([]Rec, 0, len(Catalog))
	for _, e := range Catalog {
		r := Rec{Entry: e}
		switch {
		case vramMB > 0 && e.MinVRAMGiB <= vramGiB:
			r.Fit = "gpu"
			r.Why = fmt.Sprintf("fits %.1f GiB VRAM with ~%dk context", vramGiB, e.Context/1000)
		case ramGiB > 0 && e.MinRAMGiB <= ramGiB:
			r.Fit = "cpu"
			r.Why = fmt.Sprintf("larger than %.1f GiB VRAM; runs on CPU (needs ~%.1f GiB RAM)", vramGiB, e.MinRAMGiB)
		default:
			r.Fit = "too-big"
			r.Why = fmt.Sprintf("needs ~%.1f GiB VRAM or ~%.1f GiB RAM", e.MinVRAMGiB, e.MinRAMGiB)
		}
		recs = append(recs, r)
	}
	rank := map[string]int{"gpu": 0, "cpu": 1, "too-big": 2}
	sort.SliceStable(recs, func(i, j int) bool {
		if rank[recs[i].Fit] != rank[recs[j].Fit] {
			return rank[recs[i].Fit] < rank[recs[j].Fit]
		}
		return recs[i].Entry.ParamsB > recs[j].Entry.ParamsB
	})
	return recs
}
