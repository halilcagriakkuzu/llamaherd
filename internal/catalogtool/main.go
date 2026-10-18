// Command catalogtool prints VRAM/RAM requirements for candidate catalog
// entries (hf/catalog.go) from the models' real GGUF headers.
//
// For every repo it downloads only the first bytes of one file over HTTP Range
// (the header holds all architecture metadata), then reuses that header for every
// quant of the repo: it writes a sparse file of each quant's true size and runs
// control.EstimateVram on it with the catalog's reference setup (51200 context,
// q8_0 KV cache, all layers on the GPU). Run: go run ./internal/catalogtool
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"llamaherd/control"
	"llamaherd/schema"
	"llamaherd/store"
)

type family struct {
	Repo   string
	Files  []string // single-file quants to evaluate
	MMProj string   // optional vision projector in the same repo
}

var families = []family{
	{"ornith-ai/Ornith-1.5-9B-GGUF", []string{"Ornith-1.5-9B-Q4_K_M.gguf", "Ornith-1.5-9B-Q6_K.gguf", "Ornith-1.5-9B-Q8_0.gguf"}, "mmproj-Ornith-1.5-9B-BF16.gguf"},
	{"unsloth/Qwen3.8-27B-GGUF", []string{"Qwen3.8-27B-UD-IQ2_XXS.gguf", "Qwen3.8-27B-UD-IQ3_XXS.gguf", "Qwen3.8-27B-UD-IQ4_XS.gguf", "Qwen3.8-27B-UD-Q4_K_XL.gguf", "Qwen3.8-27B-UD-Q6_K.gguf", "Qwen3.8-27B-Q8_0.gguf"}, "mmproj-F16.gguf"},
	{"unsloth/gpt-oss-20b-GGUF", []string{"gpt-oss-20b-Q4_K_M.gguf", "gpt-oss-20b-F16.gguf"}, ""},
	{"unsloth/Qwen3-Coder-Next-GGUF", []string{"Qwen3-Coder-Next-UD-IQ1_M.gguf", "Qwen3-Coder-Next-UD-IQ2_M.gguf", "Qwen3-Coder-Next-UD-Q3_K_XL.gguf", "Qwen3-Coder-Next-UD-IQ4_XS.gguf"}, ""},
	{"unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF", []string{"Qwen3-Coder-30B-A3B-Instruct-UD-IQ1_M.gguf", "Qwen3-Coder-30B-A3B-Instruct-UD-Q2_K_XL.gguf", "Qwen3-Coder-30B-A3B-Instruct-UD-Q3_K_XL.gguf", "Qwen3-Coder-30B-A3B-Instruct-UD-Q4_K_XL.gguf", "Qwen3-Coder-30B-A3B-Instruct-Q6_K.gguf"}, ""},
	{"unsloth/Qwen3.5-9B-GGUF", []string{"Qwen3.5-9B-UD-IQ2_M.gguf", "Qwen3.5-9B-UD-Q3_K_XL.gguf", "Qwen3.5-9B-Q4_K_M.gguf", "Qwen3.5-9B-UD-Q5_K_XL.gguf", "Qwen3.5-9B-UD-Q6_K_XL.gguf", "Qwen3.5-9B-Q8_0.gguf"}, "mmproj-F16.gguf"},
}

func sizes(repo string) map[string]int64 {
	resp, err := http.Get("https://huggingface.co/api/models/" + repo + "/tree/main?recursive=true")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var tree []struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		panic(err)
	}
	m := map[string]int64{}
	for _, t := range tree {
		m[t.Path] = t.Size
	}
	return m
}

// header downloads the first n MiB of a file.
func header(repo, file string, mib int) []byte {
	req, _ := http.NewRequest("GET", "https://huggingface.co/"+repo+"/resolve/main/"+file, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", mib<<20-1))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b
}

func sparse(path string, hdr []byte, size int64) string {
	if err := os.WriteFile(path, hdr, 0o644); err != nil {
		panic(err)
	}
	if err := os.Truncate(path, size); err != nil {
		panic(err)
	}
	return path
}

func ceilHalf(x float64) float64 { return math.Ceil(x*2) / 2 }

// extra is what the catalog stores besides the sizes: how fast the KV cache
// grows and the model's trained context, so the app can tell how much context a
// card can hold.
type extra struct {
	KVMiBPer1k     float64 `json:"kvMiBPer1k"`
	TrainedContext int     `json:"trainedContext"`
}

func main() {
	extras := map[string]extra{}
	dir, _ := os.MkdirTemp("", "catalogtool")
	defer os.RemoveAll(dir)
	restore := control.SetTotalVramForTesting(0)
	defer restore()

	for _, f := range families {
		sz := sizes(f.Repo)
		// the smallest listed quant carries the header we reuse
		first := f.Files[0]
		var info store.GGUFInfo
		var hdr []byte
		for _, mib := range []int{16, 48, 128} {
			hdr = header(f.Repo, first, mib)
			p := filepath.Join(dir, "probe-"+strings.ReplaceAll(f.Repo, "/", "_")+fmt.Sprint(mib)+".gguf") // unique: ReadGGUFInfo caches by path
			sparse(p, hdr, sz[first])
			var err error
			if info, err = store.ReadGGUFInfo(p); err == nil && info.BlockCount > 0 {
				break
			}
			os.Remove(p)
		}
		fmt.Printf("\n== %s  arch=%s layers=%d kvHeads=%d keyLen=%d fullAttn=%d interval=%d nextN=%d\n",
			f.Repo, info.Arch, info.BlockCount, info.HeadCountKV, info.KeyLength,
			info.FullAttentionLayers(), info.FullAttentionInterval, info.NextNPredictLayers)
		var mm string
		if f.MMProj != "" {
			mm = sparse(filepath.Join(dir, "mmproj.gguf"), []byte("GGUF"), sz[f.MMProj])
		}
		names := append([]string(nil), f.Files...)
		sort.Strings(names)
		for _, file := range f.Files {
			p := sparse(filepath.Join(dir, strings.ReplaceAll(file, "/", "_")), hdr, sz[file])
			cfg := schema.Config{Basic: schema.Basic{Model: p, Context: 51200, GPULayers: "all", KCache: "q8_0", VCache: "q8_0"}}
			cfg.Advanced.MMProj = mm
			e := control.EstimateVram(cfg)
			total := float64(sz[file])
			if f.MMProj != "" {
				total += float64(sz[f.MMProj])
			}
			small := cfg
			small.Basic.Context = 1024
			es := control.EstimateVram(small)
			kv := (e.KVKMB + e.KVVMB - es.KVKMB - es.KVVMB) / float64(51200-1024) * 1024
			extras[file] = extra{KVMiBPer1k: math.Round(kv*100) / 100, TrainedContext: info.ContextLength}
			needVRAM := ceilHalf(e.TotalMB / 1024 / 0.9)
			needRAM := ceilHalf(e.TotalMB/1024 + 1.5)
			fmt.Printf("  %-52s file %6.2f GiB  total dl %6.2f  est %6.2f GiB (kv %.2f+%.2f ssm %.2f compute %.2f)  -> minVRAM %.1f  minRAM %.1f  bytes %d\n",
				file, float64(sz[file])/(1<<30), total/(1<<30), e.TotalMB/1024, e.KVKMB/1024, e.KVVMB/1024, e.SSMMB/1024, e.ComputeMB/1024, needVRAM, needRAM, int64(total))
		}
	}
	b, _ := json.MarshalIndent(extras, "", "  ")
	fmt.Printf("\n---JSON---\n%s\n", b)
}
