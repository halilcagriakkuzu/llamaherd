package hf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchModelInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/models/acme/open":
			_, _ = w.Write([]byte(`{"id":"acme/open","gated":false,"lastModified":"2026-09-01T10:00:00.000Z",
				"tags":["gguf","license:apache-2.0","text-generation"],"cardData":{"license":"apache-2.0"}}`))
		case "/api/models/acme/gated":
			_, _ = w.Write([]byte(`{"id":"acme/gated","gated":"manual","tags":["license:other"],
				"cardData":{"license":["other"],"license_name":"acme-community"}}`))
		case "/api/models/acme/tagonly":
			_, _ = w.Write([]byte(`{"id":"acme/tagonly","gated":"auto","tags":["license:mit"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("HF_ENDPOINT", srv.URL)
	ctx := context.Background()

	open, err := FetchModelInfo(ctx, "acme/open", "")
	if err != nil {
		t.Fatal(err)
	}
	if open.License != "apache-2.0" || open.Gated != "no" || open.IsGated() {
		t.Errorf("open = %+v", open)
	}

	gated, err := FetchModelInfo(ctx, "acme/gated", "")
	if err != nil {
		t.Fatal(err)
	}
	if !gated.IsGated() || gated.Gated != "manual" || gated.License != "other" || gated.LicenseName != "acme-community" {
		t.Errorf("gated = %+v", gated)
	}

	tagOnly, err := FetchModelInfo(ctx, "acme/tagonly", "")
	if err != nil {
		t.Fatal(err)
	}
	if tagOnly.License != "mit" || tagOnly.Gated != "auto" {
		t.Errorf("tag-only = %+v", tagOnly)
	}

	if _, err := FetchModelInfo(ctx, "acme/missing", ""); err == nil {
		t.Error("404 must be an error")
	}
	if _, err := FetchModelInfo(ctx, "not-a-repo", ""); err == nil {
		t.Error("a ref without owner/name must be rejected")
	}
}
