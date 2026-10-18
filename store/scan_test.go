package store

import (
	"os"
	"testing"

	"llamaherd/dummy"
	"llamaherd/settings"
)

func TestScanModels(t *testing.T) {
	dir := settings.ModelsDir()
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("models dir not present: %s", dir)
	}
	models := ScanModels(dir)
	if len(models) == 0 {
		t.Skipf("no *.gguf found under %s", dir)
	}
	known := 0
	mmproj := 0
	for _, m := range models {
		if m.IsMMProj {
			mmproj++
			continue
		}
		for _, p := range dummy.Presets {
			for _, pr := range p {
				if settings.ResolveModelPath(pr.Basic.Model) == m.Path {
					known++
				}
			}
		}
	}
	t.Logf("scanned %d entries (%d loadable matching built-in presets, %d mmproj)", len(models), known, mmproj)
	if known == 0 {
		t.Errorf("no scanned model matches the built-in preset paths")
	}
}

func TestDerivedAlias(t *testing.T) {
	got := DerivedAlias("/a/b/Qwen3.8-27B-UD-IQ4_XS.gguf")
	want := "qwen3-8-27b-ud-iq4_xs"
	if got != want {
		t.Errorf("DerivedAlias = %q, want %q", got, want)
	}
}
