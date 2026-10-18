// gguf.go — a tiny GGUF metadata reader (stdlib only). The VRAM estimator
// (control/vram.go) needs just a handful of architecture fields: how many
// layers, how many KV heads, the K/V head dimensions and — for hybrid models
// like qwen35 — how often a full-attention layer occurs. Reading only those
// lets us size the KV cache correctly instead of guessing from the file size.
package store

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

// GGUF metadata value types (spec v3).
const (
	ggufUint8   = 0
	ggufInt8    = 1
	ggufUint16  = 2
	ggufInt16   = 3
	ggufUint32  = 4
	ggufInt32   = 5
	ggufFloat32 = 6
	ggufBool    = 7
	ggufString  = 8
	ggufArray   = 9
	ggufUint64  = 10
	ggufInt64   = 11
	ggufFloat64 = 12
)

// GGUFInfo is the slice of model metadata the VRAM estimator needs.
type GGUFInfo struct {
	Arch            string
	BlockCount      int // number of layers
	HeadCount       int
	HeadCountKV     int
	KeyLength       int // per-head K dimension
	ValueLength     int // per-head V dimension
	EmbeddingLength int
	// FullAttentionInterval: hybrid models (qwen35/Qwen3-Next-style) run full
	// attention only every Nth layer; the rest are SSM/recurrent. 0 or 1 means
	// every layer keeps a KV cache.
	FullAttentionInterval int
	// KVLayers > 0: per-layer head_count_kv array (LFM2-style hybrid) had this
	// many non-zero entries; only those layers keep a KV cache.
	KVLayers      int
	ContextLength int // trained context (upper bound)
	// NextNPredictLayers > 0 means the file carries an embedded MTP (NextN)
	// head, usable with --spec-type draft-mtp.
	NextNPredictLayers int
	// SSM (recurrent) state, context-independent; only used to add a small
	// fixed amount for hybrid models.
	SSMInnerSize  int
	SSMStateSize  int
	SSMConvKernel int
	FileSize      int64
}

// FullAttentionLayers returns how many layers keep a KV cache. For hybrid
// models this is block_count/full_attention_interval; otherwise every layer.
func (g GGUFInfo) FullAttentionLayers() int {
	if g.BlockCount <= 0 {
		return 0
	}
	if g.KVLayers > 0 {
		return g.KVLayers
	}
	iv := g.FullAttentionInterval
	if iv <= 1 {
		return g.BlockCount
	}
	n := g.BlockCount / iv
	if n < 1 {
		n = 1
	}
	return n
}

var ggufCache sync.Map // path -> GGUFInfo (immutable model metadata)

// ReadGGUFInfo parses a model's GGUF header/metadata (cached per path).
// Missing optional fields are derived from what is present.
func ReadGGUFInfo(path string) (GGUFInfo, error) {
	if v, ok := ggufCache.Load(path); ok {
		return v.(GGUFInfo), nil
	}
	f, err := os.Open(path)
	if err != nil {
		return GGUFInfo{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return GGUFInfo{}, err
	}
	info, err := parseGGUF(bufio.NewReaderSize(f, 1<<20))
	if err != nil {
		return GGUFInfo{}, err
	}
	if info.HeadCountKV == 0 {
		info.HeadCountKV = info.HeadCount
	}
	if (info.KeyLength == 0 || info.ValueLength == 0) && info.HeadCount > 0 && info.EmbeddingLength > 0 {
		hd := info.EmbeddingLength / info.HeadCount
		if info.KeyLength == 0 {
			info.KeyLength = hd
		}
		if info.ValueLength == 0 {
			info.ValueLength = hd
		}
	}
	info.FileSize = st.Size()
	ggufCache.Store(path, info)
	return info, nil
}

func parseGGUF(r *bufio.Reader) (GGUFInfo, error) {
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return GGUFInfo{}, err
	}
	if string(magic[:]) != "GGUF" {
		return GGUFInfo{}, fmt.Errorf("not a GGUF file")
	}
	var version uint32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return GGUFInfo{}, err
	}
	var tensorCount, kvCount uint64
	if err := binary.Read(r, binary.LittleEndian, &tensorCount); err != nil {
		return GGUFInfo{}, err
	}
	if err := binary.Read(r, binary.LittleEndian, &kvCount); err != nil {
		return GGUFInfo{}, err
	}
	if kvCount > 1<<20 {
		return GGUFInfo{}, fmt.Errorf("gguf: implausible metadata count %d", kvCount)
	}

	var info GGUFInfo
	for i := uint64(0); i < kvCount; i++ {
		key, err := ggufStringVal(r)
		if err != nil {
			return GGUFInfo{}, err
		}
		var typ uint32
		if err := binary.Read(r, binary.LittleEndian, &typ); err != nil {
			return GGUFInfo{}, err
		}
		if typ == ggufString && key == "general.architecture" {
			s, err := ggufStringVal(r)
			if err != nil {
				return GGUFInfo{}, err
			}
			info.Arch = s
			continue
		}
		if typ == ggufArray && hasSuffix(key, ".attention.head_count_kv") {
			if err := readKVHeadArray(r, &info); err != nil {
				return GGUFInfo{}, err
			}
			continue
		}
		// Numeric metadata we care about (arch prefix is stripped via suffix).
		if n, ok := ggufIntVal(r, typ); ok {
			applyGGUFKey(&info, key, int(n))
			continue
		}
		if err := ggufSkip(r, typ); err != nil {
			return GGUFInfo{}, err
		}
	}
	return info, nil
}

// applyGGUFKey records a recognised numeric metadata key (matched by suffix so
// the architecture prefix — qwen35./llama./etc. — does not matter).
func applyGGUFKey(info *GGUFInfo, key string, n int) {
	switch {
	case hasSuffix(key, ".block_count"):
		info.BlockCount = n
	case hasSuffix(key, ".attention.head_count_kv"):
		info.HeadCountKV = n
	case hasSuffix(key, ".attention.head_count"):
		info.HeadCount = n
	case hasSuffix(key, ".attention.key_length"):
		info.KeyLength = n
	case hasSuffix(key, ".attention.value_length"):
		info.ValueLength = n
	case hasSuffix(key, ".embedding_length"):
		info.EmbeddingLength = n
	case hasSuffix(key, ".full_attention_interval"):
		info.FullAttentionInterval = n
	case hasSuffix(key, ".nextn_predict_layers"):
		info.NextNPredictLayers = n
	case hasSuffix(key, ".context_length"):
		info.ContextLength = n
	case hasSuffix(key, ".ssm.inner_size"):
		info.SSMInnerSize = n
	case hasSuffix(key, ".ssm.state_size"):
		info.SSMStateSize = n
	case hasSuffix(key, ".ssm.conv_kernel"):
		info.SSMConvKernel = n
	}
}

func hasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}

// ggufIntVal reads a metadata value when its type is an integer width, and
// reports ok=false for any other type (the caller then skips it).
func ggufIntVal(r *bufio.Reader, typ uint32) (int64, bool) {
	switch typ {
	case ggufUint8:
		b, err := r.ReadByte()
		if err != nil {
			return 0, false
		}
		return int64(b), true
	case ggufInt8:
		b, err := r.ReadByte()
		if err != nil {
			return 0, false
		}
		return int64(int8(b)), true
	case ggufUint16:
		var v uint16
		if binary.Read(r, binary.LittleEndian, &v) != nil {
			return 0, true
		}
		return int64(v), true
	case ggufInt16:
		var v int16
		if binary.Read(r, binary.LittleEndian, &v) != nil {
			return 0, true
		}
		return int64(v), true
	case ggufUint32:
		var v uint32
		if binary.Read(r, binary.LittleEndian, &v) != nil {
			return 0, true
		}
		return int64(v), true
	case ggufInt32:
		var v int32
		if binary.Read(r, binary.LittleEndian, &v) != nil {
			return 0, true
		}
		return int64(v), true
	case ggufUint64:
		var v uint64
		if binary.Read(r, binary.LittleEndian, &v) != nil {
			return 0, true
		}
		return int64(v), true
	case ggufInt64:
		var v int64
		if binary.Read(r, binary.LittleEndian, &v) != nil {
			return 0, true
		}
		return v, true
	}
	return 0, false
}

func ggufStringVal(r *bufio.Reader) (string, error) {
	var n uint64
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return "", err
	}
	if n > 1<<26 {
		return "", fmt.Errorf("gguf: string of %d bytes is implausible", n)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

// ggufSkip consumes a metadata value of the given type without materialising
// it (the tokenizer token/merge arrays are hundreds of thousands of entries).
func ggufSkip(r *bufio.Reader, typ uint32) error {
	switch typ {
	case ggufUint8, ggufInt8, ggufBool:
		return skipN(r, 1)
	case ggufUint16, ggufInt16:
		return skipN(r, 2)
	case ggufUint32, ggufInt32, ggufFloat32:
		return skipN(r, 4)
	case ggufUint64, ggufInt64, ggufFloat64:
		return skipN(r, 8)
	case ggufString:
		_, err := ggufStringVal(r)
		return err
	case ggufArray:
		var elemType uint32
		if err := binary.Read(r, binary.LittleEndian, &elemType); err != nil {
			return err
		}
		var n uint64
		if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
			return err
		}
		if n > 1<<32 {
			return fmt.Errorf("gguf: array of %d elements is implausible", n)
		}
		if sz, ok := fixedSize(elemType); ok {
			return skipN(r, int64(sz)*int64(n))
		}
		if elemType == ggufString {
			for i := uint64(0); i < n; i++ {
				if _, err := ggufStringVal(r); err != nil {
					return err
				}
			}
			return nil
		}
		if elemType == ggufArray {
			for i := uint64(0); i < n; i++ {
				if err := ggufSkip(r, ggufArray); err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("gguf: unknown array element type %d", elemType)
	}
	return fmt.Errorf("gguf: unknown value type %d", typ)
}

func fixedSize(typ uint32) (int, bool) {
	switch typ {
	case ggufUint8, ggufInt8, ggufBool:
		return 1, true
	case ggufUint16, ggufInt16:
		return 2, true
	case ggufUint32, ggufInt32, ggufFloat32:
		return 4, true
	case ggufUint64, ggufInt64, ggufFloat64:
		return 8, true
	}
	return 0, false
}

func skipN(r *bufio.Reader, n int64) error {
	if n < 0 {
		return fmt.Errorf("gguf: negative skip")
	}
	for n > 0 {
		chunk := n
		if chunk > 1<<20 {
			chunk = 1 << 20
		}
		if _, err := io.CopyN(io.Discard, r, chunk); err != nil {
			return err
		}
		n -= chunk
	}
	return nil
}

// readKVHeadArray parses a per-layer head_count_kv array: layers with a zero
// entry (conv/SSM) carry no KV cache.
func readKVHeadArray(r *bufio.Reader, info *GGUFInfo) error {
	var elemType uint32
	var n uint64
	if err := binary.Read(r, binary.LittleEndian, &elemType); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return err
	}
	if n > 1<<16 {
		return fmt.Errorf("gguf: head_count_kv array of %d elements is implausible", n)
	}
	for i := uint64(0); i < n; i++ {
		v, ok := ggufIntVal(r, elemType)
		if !ok {
			return fmt.Errorf("gguf: head_count_kv element type %d unsupported", elemType)
		}
		if v > 0 {
			info.KVLayers++
			if int(v) > info.HeadCountKV {
				info.HeadCountKV = int(v)
			}
		}
	}
	return nil
}
