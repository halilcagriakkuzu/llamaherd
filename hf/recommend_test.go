package hf

import "testing"

func fams(recs []Rec) map[string]Rec {
	m := map[string]Rec{}
	for _, r := range recs {
		f := r.Entry.Family
		if f == "" {
			f = r.Entry.ID
		}
		m[f] = r
	}
	return m
}

func TestRecommendHidesWhatDoesNotFit(t *testing.T) {
	// 8 GiB card, 32 GiB RAM: Qwen3-Coder-Next (>=20 GiB weights) must not appear.
	recs := Recommend(8*1024, 32*1024)
	m := fams(recs)
	if _, ok := m["qwen3-coder-next"]; ok {
		t.Error("80B coder model must be hidden on an 8 GiB card")
	}
	if r, ok := m["qwen3.8-27b"]; ok && r.Fit == FitGPU {
		t.Error("27B cannot fit an 8 GiB card entirely")
	}
	if r, ok := m["qwen3.5-9b"]; !ok || r.Fit != FitGPU {
		t.Errorf("a 9B quant should fit an 8 GiB card: %+v", r)
	}
	for _, r := range recs {
		if r.Fit == "" {
			t.Errorf("entry without a fit: %+v", r)
		}
	}
}

func TestRecommendPicksBestQuantPerFamily(t *testing.T) {
	m := fams(Recommend(16*1024, 32*1024))
	r := m["qwen3.5-9b"]
	if r.Entry.Quant != "Q8_0" {
		t.Errorf("16 GiB should get the highest 9B quant, got %s", r.Entry.Quant)
	}
	if o := m["ornith-1.5-9b"]; o.Entry.Quant != "Q8_0" {
		t.Errorf("ornith quant = %s", o.Entry.Quant)
	}
	// one row per family
	count := map[string]int{}
	for _, rec := range Recommend(16*1024, 32*1024) {
		f := rec.Entry.Family
		if f == "" {
			f = rec.Entry.ID
		}
		count[f]++
		if count[f] > 1 {
			t.Errorf("family %s listed twice", f)
		}
	}
}

func TestRecommendHybridOffload(t *testing.T) {
	// 16 GiB card + plenty of RAM: the 80B MoE is shown as a hybrid fit, the
	// 30B-A3B Q4 class fits in VRAM.
	m := fams(Recommend(16*1024, 64*1024))
	if r, ok := m["qwen3-coder-next"]; !ok || r.Fit != FitHybrid || r.Share < 0.5 {
		t.Errorf("coder-next on 16 GiB: %+v", r)
	}
	// ... but not without the RAM to hold the spilled weights
	if _, ok := fams(Recommend(16*1024, 8*1024))["qwen3-coder-next"]; ok {
		t.Error("hybrid fit needs enough RAM for the spilled weights")
	}
}

func TestRecommendCPUOnlyCapsSize(t *testing.T) {
	m := fams(Recommend(0, 32*1024)) // no GPU
	for _, r := range m {
		if r.Fit != FitCPU {
			t.Errorf("no GPU: %s should be cpu, got %s", r.Entry.ID, r.Fit)
		}
	}
	if _, ok := m["qwen3.8-27b"]; ok {
		t.Error("a dense 27B is not a sensible CPU-only suggestion")
	}
	if len(Recommend(0, 0)) != 0 {
		t.Error("unknown hardware: nothing can be recommended")
	}
}

func TestStarterIsAlwaysRecommendable(t *testing.T) {
	if r := fams(Recommend(0, 4*1024))[Starter().ID]; r.Fit != FitCPU {
		t.Errorf("starter must run on a 4 GiB RAM machine: %+v", r)
	}
}

func TestPicksAreFewAndDistinct(t *testing.T) {
	recs := Recommend(16*1024, 32*1024)
	picks := Picks(recs)
	if len(picks) < 3 || len(picks) > 5 {
		t.Fatalf("want 3-5 picks, got %d", len(picks))
	}
	seen := map[string]bool{}
	for _, p := range picks {
		if seen[p.Rec.Entry.ID] {
			t.Errorf("%s picked twice", p.Rec.Entry.ID)
		}
		seen[p.Rec.Entry.ID] = true
		if p.Label == "" {
			t.Errorf("pick %s has no label", p.Role)
		}
	}
	if picks[0].Role != RoleQuality {
		t.Errorf("best quality must come first, got %s", picks[0].Role)
	}
	roles := map[string]Pick{}
	for _, p := range picks {
		roles[p.Role] = p
	}
	if v, ok := roles[RoleVision]; !ok || !v.Rec.Entry.Vision {
		t.Errorf("vision pick: %+v", v)
	}
	hasMTP := false
	for _, p := range picks {
		hasMTP = hasMTP || p.Rec.Entry.MTP // an MTP model may be picked under another role
	}
	if !hasMTP {
		t.Error("an MTP model should be among the picks when one fits")
	}
	if l, ok := roles[RoleLight]; ok && l.Rec.Entry.Bytes >= roles[RoleQuality].Rec.Entry.Bytes {
		t.Error("the lightweight pick should be smaller than the quality pick")
	}
}

func TestPicksOnSmallMachine(t *testing.T) {
	picks := Picks(Recommend(0, 8*1024)) // CPU only, 8 GiB
	if len(picks) == 0 {
		t.Fatal("a CPU-only machine still gets a recommendation")
	}
	for _, p := range picks {
		if p.Rec.Fit != FitCPU {
			t.Errorf("no GPU: %s is %s", p.Rec.Entry.ID, p.Rec.Fit)
		}
	}
}

func TestMaxContext(t *testing.T) {
	e := Entry{Bytes: 10 << 30, MinVRAMGiB: 12, MinRAMGiB: 12, Context: 51200, KVMiBPer1k: 20, TrainedContext: 262144}
	small := maxContext(e, 16)
	big := maxContext(e, 24)
	if small < e.Context || big <= small {
		t.Errorf("more VRAM must allow more context: %d then %d", small, big)
	}
	if big > e.TrainedContext {
		t.Errorf("context %d exceeds the trained window", big)
	}
	if maxContext(Entry{Bytes: 1, MinVRAMGiB: 1, Context: 51200}, 16) != 0 {
		t.Error("unknown KV growth means unknown max context")
	}
}

func TestBrowseListsEverythingGroupedByFamily(t *testing.T) {
	all := Browse(8*1024, 16*1024)
	if len(all) != len(Entries()) {
		t.Fatalf("browse shows %d of %d entries", len(all), len(Entries()))
	}
	none := 0
	last := ""
	closed := map[string]bool{}
	for _, r := range all {
		f := r.Entry.Family
		if f == "" {
			f = r.Entry.ID
		}
		if f != last {
			if closed[f] {
				t.Errorf("family %s is split across the list", f)
			}
			if last != "" {
				closed[last] = true
			}
			last = f
		}
		if r.Fit == FitNone {
			none++
		}
	}
	if none == 0 {
		t.Error("an 8 GiB machine must have entries that do not fit")
	}
}
