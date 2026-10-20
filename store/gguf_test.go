package store

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"llamaherd/internal/ggufgen"
)

func TestReadGGUFInfoDense(t *testing.T) {
	p, err := ggufgen.Write(filepath.Join(t.TempDir(), "dense.gguf"),
		ggufgen.Spec{Arch: "llama", Layers: 32, Heads: 32, KVHeads: 8, Embedding: 4096})
	if err != nil {
		t.Fatal(err)
	}
	info, err := ReadGGUFInfo(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Arch != "llama" || info.BlockCount != 32 || info.HeadCountKV != 8 {
		t.Fatalf("unexpected info: %+v", info)
	}
	// key/value length are derived from embedding/heads when absent.
	if info.KeyLength != 128 || info.ValueLength != 128 {
		t.Errorf("derived head dim = %d/%d, want 128/128", info.KeyLength, info.ValueLength)
	}
	if got := info.FullAttentionLayers(); got != 32 {
		t.Errorf("dense model should keep KV on all layers, got %d", got)
	}
}

func TestReadGGUFInfoHybridAndMTP(t *testing.T) {
	p, _ := ggufgen.Write(filepath.Join(t.TempDir(), "hyb.gguf"), ggufgen.Spec{
		Arch: "qwen35", Layers: 64, Heads: 24, KVHeads: 4, KeyLen: 256, ValueLen: 256,
		FullAttentionInterval: 4, NextNPredictLayers: 1,
	})
	info, err := ReadGGUFInfo(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.FullAttentionLayers(); got != 16 {
		t.Errorf("hybrid attention layers = %d, want 16", got)
	}
	if info.NextNPredictLayers != 1 {
		t.Errorf("MTP head not detected: %+v", info)
	}
}

func TestReadGGUFInfoErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadGGUFInfo(filepath.Join(dir, "missing.gguf")); err == nil {
		t.Error("missing file should error")
	}
	bad := filepath.Join(dir, "bad.gguf")
	os.WriteFile(bad, []byte("this is not a gguf file at all"), 0o644)
	if _, err := ReadGGUFInfo(bad); err == nil {
		t.Error("non-GGUF file should error")
	}
	trunc := filepath.Join(dir, "trunc.gguf")
	os.WriteFile(trunc, ggufgen.Bytes(ggufgen.Spec{Layers: 8, Heads: 8, Embedding: 64})[:20], 0o644)
	if _, err := ReadGGUFInfo(trunc); err == nil {
		t.Error("truncated GGUF should error")
	}
}

// FuzzParseGGUF makes sure untrusted GGUF headers can never panic the reader.
// The seed corpus contains a valid header plus a few truncated/mutated shapes.
func FuzzParseGGUF(f *testing.F) {
	valid := ggufgen.Bytes(ggufgen.Spec{Arch: "llama", Layers: 32, Heads: 8, KVHeads: 8, Embedding: 512, ContextLength: 32768})
	f.Add(valid)
	f.Add([]byte("GGUF"))
	f.Add(append([]byte("GGUF"), valid[4:20]...))
	f.Add(append([]byte("GGUF"), bytes.Repeat([]byte{0xff}, 64)...))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = parseGGUF(bufio.NewReader(bytes.NewReader(data)))
	})
}
