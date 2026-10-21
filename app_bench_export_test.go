package main

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"llamaherd/schema"
	"llamaherd/settings"
	"llamaherd/store"
)

func TestExportBenchmarksCSVAndJSON(t *testing.T) {
	a, _ := libApp(t)
	a.bench = []schema.Benchmark{
		{ModelID: "m1", Model: "Model, one", Preset: "default", Fingerprint: "abc", At: 1760000000,
			Context: 32768, GPULayers: "all", GenTokS: 42.5, PromptTokS: 600, PeakVRAMMB: 15000,
			PromptEvalMS: 350, SlotsActive: 1, SlotsBusy: 1, QueueDepth: 2, ContextTokens: 214, KVCacheRatio: 0.25},
	}
	csvBytes, err := a.ExportBenchmarks("csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(csvBytes))).ReadAll()
	if err != nil {
		t.Fatalf("CSV does not parse: %v\n%s", err, csvBytes)
	}
	if len(rows) != 2 || len(rows[0]) != 21 {
		t.Fatalf("rows = %d cols = %d", len(rows), len(rows[0]))
	}
	if rows[1][1] != "Model, one" {
		t.Errorf("CSV quoting failed: %q", rows[1][1])
	}
	if !strings.Contains(rows[0][0], "modelId") || rows[0][6] != "when" || rows[0][5] != "at" {
		t.Errorf("header = %v", rows[0])
	}

	jsonBytes, err := a.ExportBenchmarks("json")
	if err != nil {
		t.Fatal(err)
	}
	var list []schema.Benchmark
	if err := json.Unmarshal(jsonBytes, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].QueueDepth != 2 || list[0].KVCacheRatio != 0.25 {
		t.Fatalf("JSON round-trip = %+v", list)
	}
	// The exported JSON is also valid input for the store's loader.
	if err := store.SaveBenchmark(settings.BenchmarksDir(), list[0]); err != nil {
		t.Fatal(err)
	}
	loaded := store.LoadBenchmarks(settings.BenchmarksDir(), "m1")
	if len(loaded) != 1 || loaded[0].PromptEvalMS != 350 {
		t.Fatalf("store round-trip = %+v", loaded)
	}
	if _, err := a.ExportBenchmarks("xml"); err == nil {
		t.Error("an unknown format must fail")
	}
}
