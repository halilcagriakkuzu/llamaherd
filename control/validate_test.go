package control

import (
	"path/filepath"
	"strings"
	"testing"

	"llamaherd/internal/ggufgen"
)

func TestCheckConfigFindsIssues(t *testing.T) {
	p, err := ggufgen.Write(filepath.Join(t.TempDir(), "m.gguf"), ggufgen.Spec{
		Arch: "qwen35", Layers: 32, Heads: 8, KVHeads: 2, KeyLen: 128, ValueLen: 128,
		ContextLength: 8192, PadBytes: 1 << 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	fakeGPU(t, 16384)

	c := baseCfg(p)
	c.Basic.Context = 16384  // above the trained 8192
	c.Basic.GPULayers = "40" // above 32
	c.Basic.Port = 0
	issues, est := CheckConfig(c)
	if est.TotalMB <= 0 {
		t.Fatalf("estimate not computed: %+v", est)
	}
	fields := map[string]string{}
	for _, is := range issues {
		fields[is.Field] = is.Severity
	}
	for _, want := range []string{"context", "gpuLayers", "port"} {
		if _, ok := fields[want]; !ok {
			t.Errorf("missing issue for %q: %+v", want, issues)
		}
	}
	if fields["port"] != "error" {
		t.Errorf("port severity = %q, want error", fields["port"])
	}

	// A sane config must not produce errors.
	c = baseCfg(p)
	c.Basic.Context = 4096
	c.Basic.Port = 8090
	issues, _ = CheckConfig(c)
	for _, is := range issues {
		if is.Severity == "error" {
			t.Errorf("unexpected error for sane config: %+v", is)
		}
	}
}

func TestCheckConfigVRAMError(t *testing.T) {
	p, err := ggufgen.Write(filepath.Join(t.TempDir(), "big.gguf"), ggufgen.Spec{
		Arch: "llama", Layers: 8, Heads: 8, KVHeads: 2, KeyLen: 128, ValueLen: 128,
		ContextLength: 32768, PadBytes: 10 << 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	fakeGPU(t, 4096)
	c := baseCfg(p)
	issues, est := CheckConfig(c)
	if est.Fits {
		t.Fatal("expected the estimate not to fit")
	}
	found := false
	for _, is := range issues {
		if is.Field == "context" && is.Severity == "error" && strings.Contains(is.Message, "90%") {
			found = true
		}
	}
	if !found {
		t.Errorf("missing blocking issue: %+v", issues)
	}
}
