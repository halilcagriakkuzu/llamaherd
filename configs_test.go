package main

import (
	"testing"

	"llamaherd/control"
)

// TestAllPresetsFitBudget is the end-to-end guard check: it builds the real
// preset table (built-in presets + configs/<alias>.json on disk + synthesized
// defaultPreset for scanned models) and fails if any preset's predicted GPU
// footprint exceeds the VRAM budget. This is what keeps the shipped configs
// under the 90% rule.
func TestAllPresetsFitBudget(t *testing.T) {
	a := NewApp()
	a.initBackend()
	a.initModels()
	a.initPresets()

	if control.TotalVramMB() <= 0 {
		t.Skip("no GPU visible")
	}
	if len(a.presets) == 0 {
		t.Fatal("no presets built")
	}
	checked := 0
	for alias, list := range a.presets {
		for _, p := range list {
			checked++
			e := control.EstimateVram(p.AsConfig())
			t.Logf("%-32s %-12s ngl=%-4s | %.1f/%.1f GiB (%.0f%%) fits=%v",
				alias, p.Name, p.Basic.GPULayers, e.TotalMB/1024,
				float64(e.TotalVram)/1024, e.GPUPercent, e.Fits)
			if !e.Fits {
				t.Errorf("%s/%s exceeds the VRAM budget: %s", alias, p.Name, e.Warn)
			}
		}
	}
	if checked == 0 {
		t.Fatal("checked no presets")
	}
}
