// Package hf downloads GGUF models from the Hugging Face Hub and recommends
// models that fit this machine's hardware. It talks to the public Hub API
// (api/models/<repo>/tree) and the resolve endpoint directly: no Python,
// huggingface-cli or `hf` binary is required, so it works identically in the
// GUI, the CLI and the headless daemon.
//
// Downloads are resumable (HTTP Range), verified against the Hub's size and
// sha256 (LFS oid), and never leave a partial file behind on failure.
package hf

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"llamaherd/settings"
)

// Entry is one curated, downloadable model file. File is the exact path inside
// the Hub repository. MMProj, when set, is the matching vision projector file
// (same repo). Vision/MTP flag capabilities up front; a pasted repo outside the
// catalog has them auto-detected from the downloaded file (GGUF header + sidecar
// names).
type Entry struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Repo       string  `json:"repo"`
	File       string  `json:"file"`
	MMProj     string  `json:"mmproj,omitempty"`
	ParamsB    float64 `json:"paramsB"`
	Quant      string  `json:"quant"`
	Bytes      int64   `json:"bytes"`      // approximate total download size
	MinVRAMGiB float64 `json:"minVramGiB"` // recommended VRAM with ~50k context
	MinRAMGiB  float64 `json:"minRamGiB"`  // RAM when running on CPU
	Vision     bool    `json:"vision"`
	MTP        bool    `json:"mtp,omitempty"` // ships an embedded MTP (NextN) head
	Context    int     `json:"context"`       // recommended context window
	Note       string  `json:"note"`
	Starter    bool    `json:"starter"` // the small default offered on first run
	// Family groups quants of one model; Recommend shows only the best quant of a
	// family that fits (empty = its own family).
	Family string `json:"family,omitempty"`
	// MoE marks mixture-of-experts models: few active parameters, so they stay
	// usable when most weights sit in system RAM.
	MoE bool `json:"moe,omitempty"`
	// KVMiBPer1k is how much the q8_0 KV cache grows per 1000 tokens of context
	// and TrainedContext the model's trained window; together they tell how much
	// context a card can hold (0 = unknown).
	KVMiBPer1k     float64 `json:"kvMiBPer1k,omitempty"`
	TrainedContext int     `json:"trainedContext,omitempty"`
}

// defaultContext is the ~50k context every curated entry targets.
const defaultContext = 51200

// CatalogSchema is the newest catalog file format this build understands.
const CatalogSchema = 1

// DefaultCatalogURL is where the app looks for a newer catalog (the file in the
// repository's main branch), so the model list can be updated without a release.
// LLAMAHERD_CATALOG_URL overrides it ("off" disables the network refresh).
const DefaultCatalogURL = "https://raw.githubusercontent.com/halilcagriakkuzu/llamaherd/main/hf/catalog.json"

//go:embed catalog.json
var embeddedCatalog []byte

// CatalogInfo says where the active catalog came from.
type CatalogInfo struct {
	Source  string `json:"source"` // "embedded" | "remote" | "cache"
	Updated string `json:"updated"`
	Count   int    `json:"count"`
}

var cat struct {
	mu      sync.RWMutex
	entries []Entry
	info    CatalogInfo
}

func init() {
	entries, updated, _, err := parseCatalog(embeddedCatalog)
	if err != nil {
		panic("hf: embedded catalog is invalid: " + err.Error()) // guarded by TestEmbeddedCatalogValid
	}
	cat.entries = entries
	cat.info = CatalogInfo{Source: "embedded", Updated: updated, Count: len(entries)}
}

// Entries returns a copy of the active catalog.
func Entries() []Entry {
	cat.mu.RLock()
	defer cat.mu.RUnlock()
	return append([]Entry(nil), cat.entries...)
}

// Info describes the active catalog.
func Info() CatalogInfo {
	cat.mu.RLock()
	defer cat.mu.RUnlock()
	return cat.info
}

type catalogFile struct {
	Schema  int     `json:"schema"`
	Updated string  `json:"updated"`
	Entries []Entry `json:"entries"`
}

// parseCatalog decodes and validates a catalog file. Invalid entries are
// skipped (reported in skipped) so one bad row never hides the whole list; an
// unreadable file, a newer schema or no valid entry is an error.
func parseCatalog(data []byte) (entries []Entry, updated string, skipped []string, err error) {
	var f catalogFile
	if err = json.Unmarshal(data, &f); err != nil {
		return nil, "", nil, fmt.Errorf("catalog: %w", err)
	}
	if f.Schema < 1 || f.Schema > CatalogSchema {
		return nil, "", nil, fmt.Errorf("catalog: unsupported schema %d (this build reads up to %d)", f.Schema, CatalogSchema)
	}
	seen := map[string]bool{}
	for _, e := range f.Entries {
		if msg := validEntry(e, seen); msg != "" {
			skipped = append(skipped, fmt.Sprintf("%s: %s", e.ID, msg))
			continue
		}
		seen[e.ID] = true
		if e.Context == 0 {
			e.Context = defaultContext
		}
		entries = append(entries, e)
	}
	if len(entries) == 0 {
		return nil, "", skipped, fmt.Errorf("catalog: no valid entries")
	}
	return entries, f.Updated, skipped, nil
}

func validEntry(e Entry, seen map[string]bool) string {
	switch {
	case e.ID == "" || seen[e.ID]:
		return "missing or duplicate id"
	case !strings.Contains(e.Repo, "/") || strings.ContainsAny(e.Repo, " \t"):
		return "repo must be owner/name"
	case !strings.HasSuffix(strings.ToLower(e.File), ".gguf"):
		return "file must be a .gguf path"
	case e.Bytes <= 0 || e.ParamsB <= 0 || e.MinVRAMGiB <= 0 || e.MinRAMGiB <= 0:
		return "bytes, paramsB, minVramGiB and minRamGiB must be positive"
	}
	return ""
}

// CachePath is where the last downloaded catalog is kept.
func CachePath() string { return filepath.Join(settings.DataDir(), "catalog.json") }

// CatalogURL returns the remote catalog location ("" when disabled).
func CatalogURL() string {
	v := strings.TrimSpace(os.Getenv("LLAMAHERD_CATALOG_URL"))
	switch strings.ToLower(v) {
	case "off", "none", "disabled":
		return ""
	case "":
		return DefaultCatalogURL
	}
	return v
}

// LoadCachedCatalog activates the cached remote catalog if it is valid and not
// older than the embedded one. Safe to call at startup without network.
func LoadCachedCatalog() {
	data, err := os.ReadFile(CachePath())
	if err != nil {
		return
	}
	entries, updated, _, err := parseCatalog(data)
	if err != nil || updated < Info().Updated {
		return
	}
	cat.mu.Lock()
	cat.entries = entries
	cat.info = CatalogInfo{Source: "cache", Updated: updated, Count: len(entries)}
	cat.mu.Unlock()
}

// RefreshCatalog downloads the remote catalog, activates it when valid and not
// older than the active one, and caches it. A failure leaves the active catalog
// untouched. It reports whether the active catalog changed.
func RefreshCatalog(ctx context.Context) (bool, error) {
	url := CatalogURL()
	if url == "" {
		return false, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	cl := &http.Client{Timeout: 10 * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("catalog: %s returned %s", url, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return false, err
	}
	entries, updated, _, err := parseCatalog(data)
	if err != nil {
		return false, err
	}
	if updated < Info().Updated {
		return false, nil // never move backwards
	}
	cur := Info()
	changed := cur.Source != "remote" && cur.Source != "cache" || cur.Updated != updated || cur.Count != len(entries)
	cat.mu.Lock()
	cat.entries = entries
	cat.info = CatalogInfo{Source: "remote", Updated: updated, Count: len(entries)}
	cat.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(CachePath()), 0o755); err == nil {
		tmp := CachePath() + ".tmp"
		if os.WriteFile(tmp, data, 0o644) == nil {
			_ = os.Rename(tmp, CachePath())
		}
	}
	return changed, nil
}

// Starter returns the small default model offered during first run.
func Starter() Entry {
	es := Entries()
	for _, e := range es {
		if e.Starter {
			return e
		}
	}
	return es[0]
}

// ByID returns the catalog entry with the given id.
func ByID(id string) (Entry, bool) {
	for _, e := range Entries() {
		if e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}
