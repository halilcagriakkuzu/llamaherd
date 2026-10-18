package hf

import "testing"

func TestQuantOf(t *testing.T) {
	for in, want := range map[string]string{
		"Qwen3.8-27B-UD-Q4_K_XL.gguf":               "UD-Q4_K_XL",
		"Ornith-1.5-9B-Q8_0.gguf":                   "Q8_0",
		"gpt-oss-20b-F16.gguf":                      "F16",
		"Qwen3-Coder-Next-UD-IQ1_M.gguf":            "UD-IQ1_M",
		"BF16/Qwen3.8-27B-BF16-00001-of-00002.gguf": "BF16",
		"Model-MXFP4_MOE.gguf":                      "MXFP4_MOE",
		"nothing-here.gguf":                         "",
	} {
		if got := QuantOf(in); got != want {
			t.Errorf("QuantOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClassifyRepo(t *testing.T) {
	tree := []TreeFile{
		{Path: "README.md", Size: 5},
		{Path: "Model-Q4_K_M.gguf", Size: 400},
		{Path: "Model-Q8_0.gguf", Size: 900},
		{Path: "Model-IQ2_M.gguf", Size: 200},
		{Path: "mmproj-F16.gguf", Size: 50},
		{Path: "mmproj-BF16.gguf", Size: 60},
		{Path: "MTP/mtp-Model-Q4_0.gguf", Size: 30},
		{Path: "imatrix_unsloth.gguf", Size: 1},
		{Path: "BF16/Model-BF16-00001-of-00002.gguf", Size: 1000},
		{Path: "BF16/Model-BF16-00002-of-00002.gguf", Size: 500},
	}
	l := classify("o/r", tree)
	if len(l.Models) != 4 {
		t.Fatalf("models = %+v", l.Models)
	}
	if l.Models[0].Quant != "IQ2_M" || l.Models[1].Quant != "Q4_K_M" || l.Models[2].Quant != "Q8_0" {
		t.Errorf("not sorted smallest first: %+v", l.Models)
	}
	shard := l.Models[3]
	if shard.Parts != 2 || shard.Size != 1500 || shard.Name != "Model-BF16.gguf" {
		t.Errorf("shard set = %+v", shard)
	}
	if len(l.MMProjs) != 2 || l.MMProjs[0].Name != "mmproj-F16.gguf" {
		t.Errorf("mmprojs = %+v", l.MMProjs)
	}
	if len(l.MTPDrafts) != 1 || l.MTPDrafts[0].Size != 30 {
		t.Errorf("mtp drafts = %+v", l.MTPDrafts)
	}
}

func TestNoneSentinel(t *testing.T) {
	tree := []TreeFile{{Path: "m.gguf", Size: 1}, {Path: "mmproj-F16.gguf", Size: 1}, {Path: "MTP/mtp-m.gguf", Size: 1}}
	if _, ok := ResolveMMProj(tree, "none"); ok {
		t.Error("none must skip the vision projector")
	}
	if _, ok := ResolveMMProj(tree, ""); !ok {
		t.Error("empty = use the repo's projector")
	}
	if _, ok := ResolveMTPDraft(tree, "None"); ok {
		t.Error("none must skip the MTP draft")
	}
	if !IsModelFile("m.gguf") || IsModelFile("MTP/mtp-m.gguf") {
		t.Error("an MTP sidecar is not a model")
	}
}
