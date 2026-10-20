// Package ggufgen writes tiny synthetic GGUF files (header + metadata only)
// so tests can exercise GGUF parsing, model scans and the VRAM estimator
// without real multi-gigabyte models or a GPU.
package ggufgen

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
)

// Spec describes the metadata to embed. Arch defaults to "llama".
type Spec struct {
	Arch                  string
	Layers                int
	Heads                 int
	KVHeads               int
	KeyLen                int
	ValueLen              int
	Embedding             int
	FullAttentionInterval int
	NextNPredictLayers    int
	ContextLength         int
	// PadBytes extends the file so it has a realistic on-disk size
	// (the VRAM estimate scales with file size); Write extends the file sparsely.
	PadBytes int
}

const (
	tU32 = 4
	tStr = 8
)

func putStr(b *bytes.Buffer, s string) {
	binary.Write(b, binary.LittleEndian, uint64(len(s)))
	b.WriteString(s)
}

// Bytes returns the encoded GGUF file.
func Bytes(s Spec) []byte {
	if s.Arch == "" {
		s.Arch = "llama"
	}
	ints := map[string]int{
		s.Arch + ".block_count":             s.Layers,
		s.Arch + ".attention.head_count":    s.Heads,
		s.Arch + ".attention.head_count_kv": s.KVHeads,
		s.Arch + ".attention.key_length":    s.KeyLen,
		s.Arch + ".attention.value_length":  s.ValueLen,
		s.Arch + ".embedding_length":        s.Embedding,
		s.Arch + ".full_attention_interval": s.FullAttentionInterval,
		s.Arch + ".nextn_predict_layers":    s.NextNPredictLayers,
		s.Arch + ".context_length":          s.ContextLength,
	}
	keys := make([]string, 0, len(ints))
	for k, v := range ints {
		if v != 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var b bytes.Buffer
	b.WriteString("GGUF")
	binary.Write(&b, binary.LittleEndian, uint32(3))
	binary.Write(&b, binary.LittleEndian, uint64(0))           // tensors
	binary.Write(&b, binary.LittleEndian, uint64(len(keys)+1)) // kv count
	putStr(&b, "general.architecture")
	binary.Write(&b, binary.LittleEndian, uint32(tStr))
	putStr(&b, s.Arch)
	for _, k := range keys {
		putStr(&b, k)
		binary.Write(&b, binary.LittleEndian, uint32(tU32))
		binary.Write(&b, binary.LittleEndian, uint32(ints[k]))
	}
	return b.Bytes()
}

// Write creates the file (and parent directories) and returns its path.
func Write(path string, s Spec) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	hdr := Bytes(s)
	if err := os.WriteFile(path, hdr, 0o644); err != nil {
		return "", err
	}
	if s.PadBytes > 0 {
		// Sparse extension: realistic st_size without writing gigabytes.
		return path, os.Truncate(path, int64(len(hdr))+int64(s.PadBytes))
	}
	return path, nil
}
