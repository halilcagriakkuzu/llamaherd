package main

// app_bench.go — stored speed benchmarks for (model, config) combinations.
// A run measures the server that is ACTUALLY loaded (runCfg/runPreset), saves
// the result under a fingerprint of that exact config, and never lets an old
// number masquerade as the current config's speed.

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"llamaherd/control"
	"llamaherd/schema"
	"llamaherd/settings"
	"llamaherd/store"
)

// RunBenchmark probes the running server with a fixed prompt and stores the
// result. The UI disables the button while it runs; the API/CLI get the result
// synchronously.
func (a *App) RunBenchmark() (schema.Benchmark, error) {
	info := a.mgr.Info()
	if info.Status != control.StatusRunning {
		return schema.Benchmark{}, fmt.Errorf("start the server before benchmarking")
	}
	a.stateMu.RLock()
	rc := a.runCfg
	preset := a.runPreset
	modelID, modelName := "", ""
	if rc != nil {
		if i := a.indexOfModelPath(rc.Basic.Model); i >= 0 {
			modelID = a.models[i].ID
			modelName = a.models[i].Display
		}
	}
	a.stateMu.RUnlock()
	if rc == nil {
		return schema.Benchmark{}, fmt.Errorf("no loaded config to benchmark")
	}
	if modelID == "" {
		modelID = store.DerivedAlias(rc.Basic.Model)
		modelName = rc.Basic.Alias
	}

	a.stateMu.Lock()
	if a.benchmarking {
		a.stateMu.Unlock()
		return schema.Benchmark{}, fmt.Errorf("a benchmark is already running")
	}
	a.benchmarking = true
	a.stateMu.Unlock()
	a.emitState()

	defer func() {
		a.stateMu.Lock()
		a.benchmarking = false
		a.stateMu.Unlock()
		a.emitState()
	}()

	// Sample peak VRAM while the request runs. Metrics only tick every 5 s,
	// which is too coarse for a 10-30 s benchmark.
	stopVram := make(chan struct{})
	vramDone := make(chan struct{})
	peak := 0
	if info.PID > 0 {
		go func() {
			defer close(vramDone)
			t := time.NewTicker(500 * time.Millisecond)
			defer t.Stop()
			for {
				if v := control.VramMB(info.PID); v > peak {
					peak = v
				}
				select {
				case <-stopVram:
					return
				case <-t.C:
				}
			}
		}()
	} else {
		close(vramDone)
	}
	addr := info.Listen
	if addr == "" {
		addr = fmt.Sprintf("%s:%d", rc.Basic.Host, rc.Basic.Port)
	}
	sample, err := control.RunBenchmark(context.Background(), addr, control.BenchOptions{})
	close(stopVram)
	<-vramDone
	if err != nil {
		a.Log("bench: FAILED — " + err.Error())
		return schema.Benchmark{}, err
	}

	b := schema.Benchmark{
		ModelID:       modelID,
		Model:         modelName,
		Alias:         rc.Basic.Alias,
		Preset:        preset,
		Fingerprint:   store.ConfigFingerprint(*rc),
		At:            time.Now().Unix(),
		Context:       rc.Basic.Context,
		GPULayers:     rc.Basic.GPULayers,
		PromptN:       sample.PromptTokens,
		PromptTokS:    sample.PromptTokS,
		GenN:          sample.GenTokens,
		GenTokS:       sample.GenTokS,
		PeakVRAMMB:    peak,
		DurationMS:    sample.DurationMS,
		PromptEvalMS:  sample.PromptEvalMS,
		SlotsActive:   sample.SlotsActive,
		SlotsBusy:     sample.SlotsBusy,
		QueueDepth:    sample.QueueDepth,
		ContextTokens: sample.ContextTokens,
		KVCacheRatio:  sample.KVCacheRatio,
	}
	if b.ContextTokens == 0 && b.KVCacheRatio > 0 && rc.Basic.Context > 0 {
		b.ContextTokens = int(b.KVCacheRatio * float64(rc.Basic.Context))
	}
	if err := store.SaveBenchmark(settings.BenchmarksDir(), b); err != nil {
		a.Log("bench: could not save the result: " + err.Error())
		return b, err
	}
	a.stateMu.Lock()
	a.reloadBenchmarksLocked()
	a.stateMu.Unlock()
	a.Log(fmt.Sprintf("bench: %s [%s] %.1f tok/s generation, %.1f tok/s prompt, peak VRAM %s",
		modelName, preset, b.GenTokS, b.PromptTokS, control.HumanMB(b.PeakVRAMMB)))
	a.notifyEvent("benchmark", "LlamaHerd: benchmark finished",
		fmt.Sprintf("%s [%s]: %.1f tok/s generation, %.1f tok/s prompt", modelName, preset, b.GenTokS, b.PromptTokS))
	a.emitState()
	return b, nil
}

// Benchmarks returns the stored results, fastest generation speed first.
func (a *App) Benchmarks() []schema.Benchmark {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return append([]schema.Benchmark(nil), a.bench...)
}

// ClearBenchmarks deletes every stored result.
func (a *App) ClearBenchmarks() error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if err := store.ClearBenchmarks(settings.BenchmarksDir()); err != nil {
		return err
	}
	a.bench = nil
	a.Log("bench: cleared stored results")
	a.emitStateLocked()
	return nil
}

// ExportBenchmarks returns the stored results as JSON (array, round-trips
// through store.LoadBenchmarks) or RFC 4180 CSV with a stable header row.
func (a *App) ExportBenchmarks(format string) ([]byte, error) {
	rows := a.Benchmarks()
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "csv":
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		w.UseCRLF = true
		header := []string{
			"modelId", "model", "alias", "preset", "fingerprint", "at", "when",
			"context", "gpuLayers", "promptN", "promptTokPerSec", "genN", "genTokPerSec",
			"peakVramMB", "durationMs", "promptEvalMs", "slotsActive", "slotsBusy",
			"queueDepth", "contextTokens", "kvCacheRatio",
		}
		if err := w.Write(header); err != nil {
			return nil, err
		}
		f := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
		for _, b := range rows {
			when := ""
			if b.At > 0 {
				when = time.Unix(b.At, 0).Format(time.RFC3339)
			}
			rec := []string{
				b.ModelID, b.Model, b.Alias, b.Preset, b.Fingerprint,
				strconv.FormatInt(b.At, 10), when,
				strconv.Itoa(b.Context), b.GPULayers,
				strconv.Itoa(b.PromptN), f(b.PromptTokS), strconv.Itoa(b.GenN), f(b.GenTokS),
				strconv.Itoa(b.PeakVRAMMB), strconv.FormatInt(b.DurationMS, 10),
				f(b.PromptEvalMS), strconv.Itoa(b.SlotsActive), strconv.Itoa(b.SlotsBusy),
				strconv.Itoa(b.QueueDepth), strconv.Itoa(b.ContextTokens), f(b.KVCacheRatio),
			}
			if err := w.Write(rec); err != nil {
				return nil, err
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "json":
		return json.MarshalIndent(rows, "", "  ")
	default:
		return nil, fmt.Errorf("unknown export format %q (use csv or json)", format)
	}
}

// BenchmarkExportText is the Wails-friendly form of ExportBenchmarks (the UI
// saves it through a Blob download).
func (a *App) BenchmarkExportText(format string) (string, error) {
	b, err := a.ExportBenchmarks(format)
	return string(b), err
}

// reloadBenchmarksLocked refreshes the in-memory table from disk (caller holds
// stateMu). Fastest generation speed first; ties by newest.
func (a *App) reloadBenchmarksLocked() {
	list := store.AllBenchmarks(settings.BenchmarksDir())
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].GenTokS == list[j].GenTokS {
			return list[i].At > list[j].At
		}
		return list[i].GenTokS > list[j].GenTokS
	})
	a.bench = list
}
