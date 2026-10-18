package control

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"llamaherd/internal/ggufgen"
	"llamaherd/schema"
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
			cfg := schema.Config{Basic: schema.Basic{
				Model: mp, Context: s.ctx, GPULayers: s.ngl, KCache: s.k, VCache: s.v,
			}}
			e := EstimateVram(cfg)
			t.Logf("%s | %-16s | %5.1f/%.1f GiB %3.0f%% | fits=%-5v | %s",
				mk, s.name, e.TotalMB/1024, float64(e.TotalVram)/1024, e.GPUPercent, e.Fits, e.Detail)
		}
	}
}

// fakeGPU makes TotalVramMB report mb for the duration of the test.
func fakeGPU(t *testing.T, mb int) {
	t.Helper()
	vramProbe = func() int { return mb }
	t.Cleanup(func() { vramProbe = nil })
}

// synthModel writes a hybrid-attention GGUF of the given on-disk size.
func synthModel(t *testing.T, padMB int) string {
	t.Helper()
	p, err := ggufgen.Write(filepath.Join(t.TempDir(), "m.gguf"), ggufgen.Spec{
		Arch: "qwen35", Layers: 64, Heads: 24, KVHeads: 4, KeyLen: 256, ValueLen: 256,
		FullAttentionInterval: 4, PadBytes: padMB << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func baseCfg(model string) schema.Config {
	return schema.Config{Basic: schema.Basic{
		Model: model, Context: 8192, GPULayers: "all", KCache: "q8_0", VCache: "q8_0",
	}}
}

func TestEstimateFitsAndBlocks(t *testing.T) {
	model := synthModel(t, 4096) // ~4 GiB of weights
	fakeGPU(t, 16384)
	e := EstimateVram(baseCfg(model))
	if !e.Fits || e.Warn != "" {
		t.Fatalf("4 GiB model should fit a 16 GiB card: %+v", e)
	}
	if e.KVKMB <= 0 || e.KVVMB <= 0 {
		t.Errorf("KV cache term missing: %+v", e)
	}

	fakeGPU(t, 4096) // same model on a 4 GiB card must be blocked
	e = EstimateVram(baseCfg(model))
	if e.Fits || e.Warn == "" {
		t.Fatalf("model larger than the budget must be blocked: %+v", e)
	}
	if !strings.Contains(e.Warn, "90%") {
		t.Errorf("warning should mention the 90%% rule: %q", e.Warn)
	}
}

func TestEstimateScalesWithContextAndCacheType(t *testing.T) {
	model := synthModel(t, 1024)
	fakeGPU(t, 24576)
	small, big := baseCfg(model), baseCfg(model)
	big.Basic.Context = 65536
	if EstimateVram(big).TotalMB <= EstimateVram(small).TotalMB {
		t.Error("larger context must raise the estimate")
	}
	q4 := big
	q4.Basic.KCache, q4.Basic.VCache = "q4_0", "q4_0"
	if EstimateVram(q4).TotalMB >= EstimateVram(big).TotalMB {
		t.Error("q4 KV cache must lower the estimate vs q8")
	}
}

func TestEstimateNoGPUNeverBlocks(t *testing.T) {
	fakeGPU(t, 0)
	e := EstimateVram(baseCfg(synthModel(t, 8192)))
	if !e.Fits || e.Warn != "" {
		t.Errorf("without a visible GPU there is nothing to guard: %+v", e)
	}
}

func TestEstimateMissingModel(t *testing.T) {
	fakeGPU(t, 16384)
	e := EstimateVram(baseCfg(filepath.Join(t.TempDir(), "gone.gguf")))
	if !strings.Contains(e.Detail, "could not be read") {
		t.Errorf("detail = %q", e.Detail)
	}
	if e := EstimateVram(schema.Config{}); e.Detail != "no model selected" {
		t.Errorf("empty model detail = %q", e.Detail)
	}
}

func TestSafeGpuLayers(t *testing.T) {
	model := synthModel(t, 6144)
	cfg := baseCfg(model)
	fakeGPU(t, 0)
	if got := SafeGpuLayers(cfg); got != -1 {
		t.Errorf("no GPU: got %d, want -1", got)
	}
	fakeGPU(t, 65536)
	if got := SafeGpuLayers(cfg); got != 64 {
		t.Errorf("huge GPU: got %d, want all 64 layers", got)
	}
	fakeGPU(t, 5000) // too small for the full model: must offload fewer layers
	got := SafeGpuLayers(cfg)
	if got < 0 || got >= 64 {
		t.Fatalf("small GPU: got %d, want 0..63", got)
	}
	cfg.Basic.GPULayers = strconv.Itoa(got)
	if !EstimateVram(cfg).Fits {
		t.Error("SafeGpuLayers result must itself fit")
	}
}

func TestBytesPerElement(t *testing.T) {
	if v, ok := BytesPerElement("f16"); !ok || v != 2 {
		t.Errorf("f16 = %v,%v", v, ok)
	}
	if v, ok := BytesPerElement("q8_0"); !ok || v <= 1 || v >= 1.2 {
		t.Errorf("q8_0 = %v,%v", v, ok)
	}
	if _, ok := BytesPerElement("bogus"); ok {
		t.Error("unknown type must be rejected")
	}
}
