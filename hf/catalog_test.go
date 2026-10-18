package hf

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"llamaherd/settings"
)

// TestEmbeddedCatalogValid guards hf/catalog.json (which the app also downloads
// from the main branch): every entry must be valid, ids unique, and quants of a
// family must carry consistent data.
func TestEmbeddedCatalogValid(t *testing.T) {
	entries, updated, skipped, err := parseCatalog(embeddedCatalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) > 0 {
		t.Fatalf("invalid entries: %v", skipped)
	}
	if len(updated) != 10 {
		t.Errorf("updated = %q, want YYYY-MM-DD", updated)
	}
	starters := 0
	for _, e := range entries {
		if e.Starter {
			starters++
		}
		if e.MinRAMGiB < float64(e.Bytes)/gib {
			t.Errorf("%s: minRamGiB %.1f is below the file size %.1f GiB", e.ID, e.MinRAMGiB, float64(e.Bytes)/gib)
		}
		if e.Quality <= 0 {
			t.Errorf("%s: set a quality score (see hf.qualityScore for the default formula)", e.ID)
		}
		if e.MMProj != "" && !e.Vision {
			t.Errorf("%s: has an mmproj but vision=false", e.ID)
		}
	}
	if starters != 1 {
		t.Errorf("want exactly one starter entry, got %d", starters)
	}
}

func TestParseCatalogRules(t *testing.T) {
	good := Entry{ID: "a", Repo: "o/r", File: "a.gguf", ParamsB: 1, Bytes: 1, MinVRAMGiB: 1, MinRAMGiB: 1}
	mk := func(schema int, es ...Entry) []byte {
		b, _ := json.Marshal(catalogFile{Schema: schema, Updated: "2030-01-01", Entries: es})
		return b
	}
	if es, _, sk, err := parseCatalog(mk(1, good)); err != nil || len(es) != 1 || len(sk) != 0 || es[0].Context != defaultContext {
		t.Fatalf("good catalog: %v %v %v", es, sk, err)
	}
	bad := good
	bad.ID = "b"
	bad.File = "notgguf.bin"
	if es, _, sk, err := parseCatalog(mk(1, good, bad)); err != nil || len(es) != 1 || len(sk) != 1 {
		t.Errorf("a bad row must be skipped, not fatal: %v %v %v", es, sk, err)
	}
	dup := good
	if _, _, sk, _ := parseCatalog(mk(1, good, dup)); len(sk) != 1 {
		t.Error("duplicate ids must be skipped")
	}
	if _, _, _, err := parseCatalog(mk(99, good)); err == nil {
		t.Error("a newer schema must be refused")
	}
	if _, _, _, err := parseCatalog(mk(1, bad)); err == nil {
		t.Error("no valid entries must be an error")
	}
	if _, _, _, err := parseCatalog([]byte("{nope")); err == nil {
		t.Error("garbage must be an error")
	}
}

// withIsolatedCatalog restores the active catalog and points the cache and URL
// at test locations.
func withIsolatedCatalog(t *testing.T) {
	t.Helper()
	cat.mu.RLock()
	saved, savedInfo := cat.entries, cat.info
	cat.mu.RUnlock()
	t.Cleanup(func() {
		cat.mu.Lock()
		cat.entries, cat.info = saved, savedInfo
		cat.mu.Unlock()
		settings.Reload()
	})
	t.Setenv(settings.EnvDataDir, t.TempDir())
	t.Setenv(settings.EnvConfig, filepath.Join(t.TempDir(), "c.json"))
	settings.Reload()
}

func TestRefreshCatalogFromRemote(t *testing.T) {
	withIsolatedCatalog(t)
	remote := catalogFile{Schema: 1, Updated: "2099-01-01", Entries: []Entry{
		{ID: "new-model", Name: "New", Repo: "o/new", File: "new.gguf", ParamsB: 3, Bytes: 2 << 30, MinVRAMGiB: 4, MinRAMGiB: 4, Starter: true},
	}}
	body, _ := json.Marshal(remote)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write(body)
	}))
	defer srv.Close()
	t.Setenv("LLAMAHERD_CATALOG_URL", srv.URL)

	changed, err := RefreshCatalog(context.Background())
	if err != nil || !changed {
		t.Fatalf("refresh = %v, %v", changed, err)
	}
	if info := Info(); info.Source != "remote" || info.Updated != "2099-01-01" || info.Count != 1 {
		t.Errorf("info = %+v", info)
	}
	if _, ok := ByID("new-model"); !ok {
		t.Error("remote entry not active")
	}
	if _, err := os.Stat(CachePath()); err != nil {
		t.Errorf("catalog not cached: %v", err)
	}

	// an older remote never replaces a newer active catalog
	old, _ := json.Marshal(catalogFile{Schema: 1, Updated: "2000-01-01", Entries: remote.Entries})
	body = old
	if changed, err := RefreshCatalog(context.Background()); err != nil || changed {
		t.Errorf("older remote: %v, %v", changed, err)
	}
	if Info().Updated != "2099-01-01" {
		t.Error("catalog moved backwards")
	}

	// a broken remote leaves the active catalog alone
	body = []byte("<html>rate limited</html>")
	if _, err := RefreshCatalog(context.Background()); err == nil {
		t.Error("garbage remote must error")
	}
	if _, ok := ByID("new-model"); !ok {
		t.Error("a failed refresh must keep the active catalog")
	}
}

func TestLoadCachedCatalogAndOffSwitch(t *testing.T) {
	withIsolatedCatalog(t)
	t.Setenv("LLAMAHERD_CATALOG_URL", "off")
	if changed, err := RefreshCatalog(context.Background()); changed || err != nil {
		t.Errorf("off: %v %v", changed, err)
	}
	if CatalogURL() != "" {
		t.Error("off must disable the URL")
	}
	t.Setenv("LLAMAHERD_CATALOG_URL", "")
	if !strings.HasPrefix(CatalogURL(), "https://raw.githubusercontent.com/") {
		t.Errorf("default URL = %q", CatalogURL())
	}

	cached, _ := json.Marshal(catalogFile{Schema: 1, Updated: "2098-05-05", Entries: []Entry{
		{ID: "cached", Name: "Cached", Repo: "o/c", File: "c.gguf", ParamsB: 1, Bytes: 1 << 30, MinVRAMGiB: 2, MinRAMGiB: 2},
	}})
	if err := os.MkdirAll(filepath.Dir(CachePath()), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(CachePath(), cached, 0o644)
	LoadCachedCatalog()
	if info := Info(); info.Source != "cache" || info.Updated != "2098-05-05" {
		t.Errorf("cache not loaded: %+v", info)
	}
}

func TestQualityFromCatalogOverridesTheFormula(t *testing.T) {
	e := Entry{ParamsB: 27, Bytes: 12 << 30, Quality: 99}
	if qualityScore(e) != 99 {
		t.Errorf("an explicit quality must win, got %v", qualityScore(e))
	}
	e.Quality = 0
	if got := qualityScore(e); got <= 0 || got >= 27 {
		t.Errorf("computed fallback out of range: %v", got)
	}
	low := Entry{ParamsB: 30, Bytes: 9 << 30}   // ~2.6 bits per weight
	high := Entry{ParamsB: 27, Bytes: 15 << 30} // ~4.8 bits per weight
	if qualityScore(low) >= qualityScore(high) {
		t.Error("a heavily quantized larger model must not outrank a healthy smaller one")
	}
}
