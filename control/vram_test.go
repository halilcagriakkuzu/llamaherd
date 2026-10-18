package control

import (
	"os"
	"testing"

	"llamaherd/dummy"
	"llamaherd/settings"
	"llamaherd/store"
)

func TestGGUFInfo(t *testing.T) {
	p := settings.ResolveModelPath("Qwen3.8-27B-Uncensored-HauhauCS-Aggressive-Q3_K_P.gguf")
	if _, err := os.Stat(p); err != nil {
		t.Skip("model not present")
	}
	info, err := store.ReadGGUFInfo(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("arch=%s layers=%d kvHeads=%d kLen=%d vLen=%d interval=%d ctx=%d fa=%d fileSize=%d",
		info.Arch, info.BlockCount, info.HeadCountKV, info.KeyLength, info.ValueLength,
		info.FullAttentionInterval, info.ContextLength, info.FullAttentionLayers(), info.FileSize)
	if info.BlockCount == 0 || info.KeyLength == 0 {
		t.Fatalf("metadata incomplete: %+v", info)
	}
	if info.FullAttentionLayers() >= info.BlockCount {
		t.Fatalf("expected hybrid attention, got fa=%d of %d", info.FullAttentionLayers(), info.BlockCount)
	}
}

func TestEstimateMatrix(t *testing.T) {
	models := map[string]string{
		"Q3_K_S   ": "unsloth/Qwen3.8-27B-GGUF/Qwen3.8-27B-Q3_K_S.gguf",
		"Q3_K_P   ": "Qwen3.8-27B-Uncensored-HauhauCS-Aggressive-Q3_K_P.gguf",
		"UD-IQ4_XS": "unsloth/Qwen3.8-27B-GGUF/Qwen3.8-27B-UD-IQ4_XS.gguf",
	}
	type sc struct {
		name string
		ctx  int
		ngl  string
		k, v string
	}
	scenarios := []sc{
		{"ref 25k q8 all", 25556, "all", "q8_0", "q8_0"},
		{"32k q4 all", 32768, "all", "q4_0", "q4_0"},
		{"65k q4 all", 65536, "all", "q4_0", "q4_0"},
		{"65k q4 ngl55", 65536, "55", "q4_0", "q4_0"},
		{"128k q4 ngl50", 131072, "50", "q4_0", "q4_0"},
	}
	if _, err := os.Stat(settings.ModelsDir()); err != nil {
		t.Skip("models not present")
	}
	for mk, rel := range models {
		mp := settings.ResolveModelPath(rel)
		if _, err := os.Stat(mp); err != nil {
			continue
		}
		for _, s := range scenarios {
			cfg := dummy.Config{Basic: dummy.Basic{
				Model: mp, Context: s.ctx, GPULayers: s.ngl, KCache: s.k, VCache: s.v,
			}}
			e := EstimateVram(cfg)
			t.Logf("%s | %-16s | %5.1f/%.1f GiB %3.0f%% | fits=%-5v | %s",
				mk, s.name, e.TotalMB/1024, float64(e.TotalVram)/1024, e.GPUPercent, e.Fits, e.Detail)
		}
	}
}
