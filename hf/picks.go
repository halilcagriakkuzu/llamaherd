package hf

// Pick roles, in display order.
const (
	RoleQuality = "quality"
	RoleVision  = "vision"
	RoleContext = "context"
	RoleMTP     = "mtp"
	RoleLight   = "light"
)

// Pick is one highlighted recommendation: why it was chosen plus the scored entry.
type Pick struct {
	Role  string `json:"role"`
	Label string `json:"label"`
	Rec   Rec    `json:"rec"`
}

var roleLabels = map[string]string{
	RoleQuality: "Best quality for this machine",
	RoleVision:  "Vision (understands images)",
	RoleContext: "Longest context",
	RoleMTP:     "MTP (faster generation)",
	RoleLight:   "Lightweight and fast",
}

// longContextFloor is the smallest maximum context worth a "longest context" pick.
const longContextFloor = 100_000

// Picks reduces a recommendation list (from Recommend) to a handful of
// highlights: the best-quality model, the best vision model, the model that can
// hold the longest context, an MTP model and a lightweight one. A model is
// never picked twice, and roles with no candidate are skipped.
func Picks(recs []Rec) []Pick {
	rank := map[string]int{FitGPU: 0, FitHybrid: 1, FitCPU: 2}
	used := map[string]bool{}
	var out []Pick

	best := func(role string, ok func(Rec) bool, better func(a, b Rec) bool) {
		var cur *Rec
		for i := range recs {
			r := recs[i]
			if used[r.Entry.ID] || !ok(r) {
				continue
			}
			if cur == nil || rank[r.Fit] < rank[cur.Fit] || (rank[r.Fit] == rank[cur.Fit] && better(r, *cur)) {
				c := r
				cur = &c
			}
		}
		if cur != nil {
			used[cur.Entry.ID] = true
			out = append(out, Pick{Role: role, Label: roleLabels[role], Rec: *cur})
		}
	}
	bigger := func(a, b Rec) bool { return qualityScore(a.Entry) > qualityScore(b.Entry) }
	any := func(Rec) bool { return true }

	best(RoleQuality, any, bigger)
	best(RoleVision, func(r Rec) bool { return r.Entry.Vision }, bigger)
	best(RoleContext, func(r Rec) bool { return r.Fit == FitGPU && r.MaxContext >= longContextFloor },
		func(a, b Rec) bool { return a.MaxContext > b.MaxContext })
	best(RoleMTP, func(r Rec) bool { return r.Entry.MTP }, bigger)
	// "Lightweight" should still be useful: the smallest model of at least 3B
	// parameters, falling back to any size when the catalog has none that fit.
	light := func(minParams float64) func(Rec) bool {
		return func(r Rec) bool {
			return (r.Fit == FitGPU || r.Fit == FitCPU) && r.Entry.ParamsB >= minParams
		}
	}
	smaller := func(a, b Rec) bool { return a.Entry.Bytes < b.Entry.Bytes }
	n := len(out)
	best(RoleLight, light(3), smaller)
	if len(out) == n {
		best(RoleLight, light(0), smaller)
	}
	return out
}

// qualityScore is the entry's Quality from the catalog, or, when that is unset,
// an estimate: parameter count scaled
// by how much precision the quant keeps. A 1-bit 30B quant must not outrank a
// healthy 27B one, so the factor falls from 1 at >= 4.5 bits per weight to 0.1 at
// <= 1.8 bits (bits per weight = file size / parameters).
func qualityScore(e Entry) float64 {
	if e.Quality > 0 {
		return e.Quality
	}
	if e.ParamsB <= 0 {
		return 0
	}
	bpw := float64(e.Bytes) * 8 / (e.ParamsB * 1e9)
	qf := (bpw - 1.5) / 3.0
	if qf > 1 {
		qf = 1
	}
	if qf < 0.1 {
		qf = 0.1
	}
	return e.ParamsB * qf
}
