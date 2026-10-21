package main

import (
	"encoding/json"
	"testing"
)

// TestSnapshotContract pins the top-level JSON contract the frontend reads.
// Renaming or dropping a key here is a breaking UI change, not a refactor.
func TestSnapshotContract(t *testing.T) {
	a, _ := libApp(t, "alpha")
	snap := a.snapshot()
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}

	required := []string{
		"status", "crashExit", "ready", "server", "models", "categories", "selectedModel",
		"alias", "presetNames", "selectedPreset", "config", "configPath",
		"dirty", "commandLine", "bootLines", "warning", "vramEstimate",
		"vramPercent", "vramFits", "validation", "llamaUpdating", "firstRun",
		"serverBin", "modelsDir", "userConfigPath", "doctor", "download",
		"metrics", "benchmarks", "benchmarking", "configFingerprint",
		"runFingerprint", "recentLog",
	}
	for _, key := range required {
		if _, ok := obj[key]; !ok {
			t.Errorf("snapshot is missing key %q", key)
		}
	}

	// The frontend indexes models/selectedModel and reads nested config parts.
	var cfg struct {
		Basic    map[string]any `json:"basic"`
		Advanced map[string]any `json:"advanced"`
	}
	if err := json.Unmarshal(obj["config"], &cfg); err != nil {
		t.Fatalf("config shape: %v", err)
	}
	if _, ok := cfg.Basic["model"]; !ok {
		t.Error("config.basic.model missing")
	}
	if _, ok := cfg.Advanced["specType"]; !ok {
		t.Error("config.advanced.specType missing")
	}
	var models []map[string]any
	if err := json.Unmarshal(obj["models"], &models); err != nil {
		t.Fatalf("models shape: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("models is empty")
	}
	for _, key := range []string{"id", "display", "path", "alias", "isMMProj", "category"} {
		if _, ok := models[0][key]; !ok {
			t.Errorf("models[0] is missing %q", key)
		}
	}
	var server map[string]any
	if err := json.Unmarshal(obj["server"], &server); err != nil {
		t.Fatalf("server shape: %v", err)
	}
	for _, key := range []string{"status", "pid", "listen", "model", "context", "tokPerSec", "vram"} {
		if _, ok := server[key]; !ok {
			t.Errorf("server is missing %q", key)
		}
	}
}
