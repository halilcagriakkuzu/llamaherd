// bench.go — persistent benchmark results: one JSON file per model under
// <data>/benchmarks/<model-id>.json. Every result carries a fingerprint of the
// exact config it measured, so editing a preset never makes an old number look
// like it belongs to the new config.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"llamaherd/schema"
)

// benchFileMax caps how many historical results are kept per model (one per
// distinct config fingerprint, newest first).
const benchFileMax = 40

// ConfigFingerprint is a stable hash of the launch-relevant config. The struct
// is marshalled as-is (field order is fixed), so equal configs always produce
// the same fingerprint and any edit produces a different one.
func ConfigFingerprint(c schema.Config) string {
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func benchPath(dir, modelID string) string {
	return filepath.Join(dir, filepath.Base(modelID+".json"))
}

// LoadBenchmarks reads one model's stored results (unsorted). A missing or
// malformed file yields an empty slice.
func LoadBenchmarks(dir, modelID string) []schema.Benchmark {
	data, err := os.ReadFile(benchPath(dir, modelID))
	if err != nil {
		return nil
	}
	var list []schema.Benchmark
	if json.Unmarshal(data, &list) != nil {
		return nil
	}
	return list
}

// SaveBenchmark inserts or replaces the result for the same model + preset +
// config fingerprint and writes the model's file.
func SaveBenchmark(dir string, b schema.Benchmark) error {
	if b.ModelID == "" {
		return os.ErrInvalid
	}
	list := LoadBenchmarks(dir, b.ModelID)
	kept := make([]schema.Benchmark, 0, len(list)+1)
	replaced := false
	for _, old := range list {
		if old.Preset == b.Preset && old.Fingerprint == b.Fingerprint {
			kept = append(kept, b)
			replaced = true
			continue
		}
		kept = append(kept, old)
	}
	if !replaced {
		kept = append(kept, b)
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].At > kept[j].At })
	if len(kept) > benchFileMax {
		kept = kept[:benchFileMax]
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(kept, "", "  ")
	if err != nil {
		return err
	}
	tmp := benchPath(dir, b.ModelID) + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, benchPath(dir, b.ModelID))
}

// AllBenchmarks reads every stored result under dir.
func AllBenchmarks(dir string) []schema.Benchmark {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []schema.Benchmark
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		id := e.Name()[:len(e.Name())-len(".json")]
		out = append(out, LoadBenchmarks(dir, id)...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At > out[j].At })
	return out
}

// ClearBenchmarks removes every stored result.
func ClearBenchmarks(dir string) error {
	ents, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// DeleteModelBenchmarks removes one model's result file (used when the model
// file is deleted).
func DeleteModelBenchmarks(dir, modelID string) error {
	err := os.Remove(benchPath(dir, modelID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
