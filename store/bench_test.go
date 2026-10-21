package store

import (
	"testing"

	"llamaherd/schema"
)

func TestBenchmarkPersistenceAndFingerprint(t *testing.T) {
	dir := t.TempDir()
	cfg := schema.Config{Basic: schema.Basic{Model: "a.gguf", Context: 32768}, Advanced: schema.Advanced{Fit: "off"}}
	fp := ConfigFingerprint(cfg)
	if fp == "" {
		t.Fatal("empty fingerprint")
	}
	if ConfigFingerprint(cfg) != fp {
		t.Error("fingerprint is not deterministic")
	}
	changed := cfg
	changed.Basic.Context = 16384
	if ConfigFingerprint(changed) == fp {
		t.Error("a config edit must change the fingerprint")
	}

	first := schema.Benchmark{ModelID: "a", Preset: "fast", Fingerprint: fp, At: 1, GenTokS: 40,
		PromptEvalMS: 123.5, SlotsActive: 2, SlotsBusy: 1, QueueDepth: 3, ContextTokens: 4096, KVCacheRatio: 0.25}
	if err := SaveBenchmark(dir, first); err != nil {
		t.Fatal(err)
	}
	// Same preset+fingerprint replaces; a new fingerprint is kept alongside.
	first.GenTokS = 42
	if err := SaveBenchmark(dir, first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.Fingerprint = ConfigFingerprint(changed)
	second.At = 2
	second.PromptEvalMS = 200
	second.QueueDepth = 0
	second.KVCacheRatio = 0.5
	if err := SaveBenchmark(dir, second); err != nil {
		t.Fatal(err)
	}
	list := LoadBenchmarks(dir, "a")
	if len(list) != 2 {
		t.Fatalf("results = %d, want 2", len(list))
	}
	if list[0].At != 2 {
		t.Errorf("newest first, got %+v", list)
	}
	if list[0].PromptEvalMS != 200 || list[0].KVCacheRatio != 0.5 || list[1].PromptEvalMS != 123.5 || list[1].QueueDepth != 3 {
		t.Errorf("metrics counters did not round-trip: %+v", list)
	}
	if all := AllBenchmarks(dir); len(all) != 2 {
		t.Errorf("AllBenchmarks = %d", len(all))
	}
	if err := DeleteModelBenchmarks(dir, "a"); err != nil {
		t.Fatal(err)
	}
	if all := AllBenchmarks(dir); len(all) != 0 {
		t.Errorf("delete left %d results", len(all))
	}
}

func TestClearBenchmarks(t *testing.T) {
	dir := t.TempDir()
	_ = SaveBenchmark(dir, schema.Benchmark{ModelID: "a", Preset: "p", Fingerprint: "x", At: 1})
	if err := ClearBenchmarks(dir); err != nil {
		t.Fatal(err)
	}
	if all := AllBenchmarks(dir); len(all) != 0 {
		t.Errorf("clear left %d results", len(all))
	}
}
