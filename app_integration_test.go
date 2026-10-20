package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"llamaherd/api"
	"llamaherd/control"
	"llamaherd/internal/ggufgen"
	"llamaherd/settings"
)

func apiCall(t *testing.T, srv *httptest.Server, method, path, body string, out any) {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}
}

// TestControlAPIEndToEnd drives the real App through the HTTP API: model scan,
// presets on disk, the VRAM pre-flight guard, and crash detection. It needs no
// GPU, no llama.cpp and no real models (synthetic GGUFs, fake server script).
func TestControlAPIEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a /bin/sh fake llama-server")
	}
	root := t.TempDir()
	models := filepath.Join(root, "models")
	fakeBin := filepath.Join(root, "llama-server")
	os.WriteFile(fakeBin, []byte("#!/bin/sh\necho fake-server-starting >&2\nexit 3\n"), 0o755)
	ggufgen.Write(filepath.Join(models, "small.gguf"),
		ggufgen.Spec{Arch: "llama", Layers: 8, Heads: 8, KVHeads: 8, Embedding: 512, PadBytes: 64 << 20})
	ggufgen.Write(filepath.Join(models, "huge.gguf"),
		ggufgen.Spec{Arch: "llama", Layers: 8, Heads: 8, KVHeads: 8, Embedding: 512, PadBytes: 2048 << 20})

	t.Cleanup(settings.Reload)
	t.Setenv(settings.EnvConfig, filepath.Join(root, "config.json"))
	t.Setenv(settings.EnvModelsDir, models)
	t.Setenv(settings.EnvDataDir, filepath.Join(root, "data"))
	// ConfigsDir/LibraryDir prefer a relative ./configs when it exists (the repo
	// has one), so pin them explicitly or the test would write into it.
	t.Setenv(settings.EnvConfigsDir, filepath.Join(root, "configs"))
	t.Setenv(settings.EnvLibraryDir, filepath.Join(root, "library"))
	t.Setenv(settings.EnvServerBin, fakeBin)
	t.Setenv(settings.EnvAPIToken, "")
	settings.Reload()

	a := NewApp()
	a.initBackend()
	a.initModels()
	a.initPresets()
	a.selModel = 0
	srv := httptest.NewServer(api.Handler(a))
	defer srv.Close()

	// models + synthesized default presets
	var ml struct{ Models []struct{ Alias string } }
	apiCall(t, srv, "GET", "/models", "", &ml)
	if len(ml.Models) != 2 {
		t.Fatalf("models = %+v, want 2", ml.Models)
	}
	var pl struct{ Presets []struct{ Name string } }
	apiCall(t, srv, "GET", "/presets?alias=small", "", &pl)
	if len(pl.Presets) != 1 || pl.Presets[0].Name != "default" {
		t.Fatalf("presets = %+v", pl.Presets)
	}

	// add a preset: persisted as configs/<alias>.json
	var ok struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	apiCall(t, srv, "POST", "/config/add",
		`{"alias":"small","name":"big-ctx","config":{"basic":{"context":65536}}}`, &ok)
	if !ok.OK {
		t.Fatalf("add failed: %s", ok.Error)
	}
	if _, err := os.Stat(filepath.Join(settings.ConfigsDir(), "small.json")); err != nil {
		t.Errorf("preset not persisted: %v", err)
	}
	apiCall(t, srv, "GET", "/presets?alias=small", "", &pl)
	if len(pl.Presets) != 2 {
		t.Errorf("presets after add = %+v", pl.Presets)
	}

	// VRAM guard: a 2 GiB model on a 1 GiB "GPU" must refuse to start.
	restore := control.SetTotalVramForTesting(1024)
	apiCall(t, srv, "POST", "/open", `{"alias":"huge"}`, &ok)
	restore()
	if ok.OK || !strings.Contains(ok.Error, "start blocked") {
		t.Fatalf("guard did not block: %+v", ok)
	}

	// No GPU: the guard stands down; the fake server exits 3 => crashed.
	restore = control.SetTotalVramForTesting(0)
	defer restore()
	apiCall(t, srv, "POST", "/open", `{"alias":"small"}`, &ok)
	if !ok.OK {
		t.Fatalf("open failed: %s", ok.Error)
	}
	var st struct {
		Status    string `json:"status"`
		CrashExit int    `json:"crashExit"`
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		apiCall(t, srv, "GET", "/status", "", &st)
		if st.Status == "crashed" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if st.Status != "crashed" || st.CrashExit != 3 {
		t.Fatalf("status = %+v, want crashed with exit 3", st)
	}
}
