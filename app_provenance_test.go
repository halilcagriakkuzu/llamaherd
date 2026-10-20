package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"llamaherd/internal/ggufgen"
	"llamaherd/settings"
	"llamaherd/store"
)

// fakeHub serves acme/open (MIT, public) and acme/gated (manual approval,
// files refused with 403) with one small .gguf each.
func fakeHub(t *testing.T) {
	t.Helper()
	body := ggufgen.Bytes(ggufgen.Spec{Layers: 4, Heads: 4, KVHeads: 4, Embedding: 256})
	sum := sha256.Sum256(body)
	tree := fmt.Sprintf(`[{"type":"file","path":"m-Q4_K_M.gguf","size":%d,"lfs":{"oid":"%s","size":%d}}]`,
		len(body), hex.EncodeToString(sum[:]), len(body))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/models/acme/open":
			_, _ = w.Write([]byte(`{"gated":false,"cardData":{"license":"mit"}}`))
		case "/api/models/acme/gated":
			_, _ = w.Write([]byte(`{"gated":"manual","cardData":{"license":"llama3"}}`))
		case "/api/models/acme/open/tree/main", "/api/models/acme/gated/tree/main":
			_, _ = w.Write([]byte(tree))
		case "/acme/open/resolve/main/m-Q4_K_M.gguf":
			_, _ = w.Write(body)
		case "/acme/gated/resolve/main/m-Q4_K_M.gguf":
			http.Error(w, "gated", http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("HF_ENDPOINT", srv.URL)
	t.Setenv("HF_TOKEN", "")
	t.Setenv("HUGGING_FACE_HUB_TOKEN", "")
	t.Setenv("HF_TOKEN_PATH", filepath.Join(t.TempDir(), "no-token"))
}

func TestDownloadRecordsProvenance(t *testing.T) {
	a, _ := libApp(t)
	fakeHub(t)
	if err := a.doDownload(context.Background(), "acme/open", "", "none", "none"); err != nil {
		t.Fatal(err)
	}
	i := a.indexOfModelPath(filepath.Join(settings.ModelsDir(), "acme/open", "m-Q4_K_M.gguf"))
	if i < 0 {
		t.Fatalf("downloaded model missing: %+v", a.models)
	}
	p := a.models[i].Provenance
	if p == nil || p.Repo != "acme/open" || p.License != "mit" || !strings.HasSuffix(p.URL, "/acme/open") {
		t.Errorf("provenance = %+v", p)
	}
	meta, _ := store.LoadMeta(settings.LibraryDir(), a.models[i].ID)
	if meta.Provenance == nil || meta.Provenance.License != "mit" {
		t.Errorf("meta provenance = %+v", meta.Provenance)
	}
}

func TestGatedDownloadExplainsLicence(t *testing.T) {
	a, _ := libApp(t)
	fakeHub(t)

	// No token: refused before any file transfer, with the licence page.
	err := a.doDownload(context.Background(), "acme/gated", "", "none", "none")
	if err == nil || !strings.Contains(err.Error(), "gated repository") ||
		!strings.Contains(err.Error(), "/acme/gated") || !strings.Contains(err.Error(), "token") {
		t.Fatalf("gated without token = %v", err)
	}

	// With a token whose account has not accepted the licence: the 403 is
	// explained instead of a bare "access denied".
	t.Setenv("HF_TOKEN", "tok")
	err = a.doDownload(context.Background(), "acme/gated", "", "none", "none")
	if err == nil || !strings.Contains(err.Error(), "accept the licence") || !strings.Contains(err.Error(), "manually") {
		t.Fatalf("gated with token = %v", err)
	}
	if len(a.models) != 0 {
		t.Errorf("a refused download must not install anything: %+v", a.models)
	}
}
