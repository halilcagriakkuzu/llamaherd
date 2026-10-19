// Package api exposes the App's control surface over HTTP so AI agents (or
// scripts) can drive the app the same way the GUI does. One source of truth:
// the GUI window reflects every CLI/API change live, and vice versa.
// BOUND TO 127.0.0.1 ONLY.
package api

import (
	"encoding/json"
	"log"
	"net/http"

	"llamaherd/schema"
	"llamaherd/settings"
)

// Control is the subset of *main.App the API needs. main.App implements it.
type Control interface {
	SnapshotJSON() ([]byte, error)
	DoctorJSON() ([]byte, error)
	DownloadJSON() ([]byte, error)
	CatalogJSON(all bool) ([]byte, error)
	StartDownload(ref, file, visionFile, mtpFile string) error
	RepoFilesJSON(ref string) ([]byte, error)
	CancelDownload() error
	SetHFToken(token string) error
	Models() []schema.Model
	Presets(alias string) []schema.Preset
	SelectModelByAlias(alias string) error
	SelectPreset(name string)
	Open(alias, preset string) error
	Stop() error
	SaveConfig()
	LoadConfig()
	AddPreset(alias, name string, cfg schema.Config) error
	UpdatePreset(alias, name string, cfg schema.Config) error
	RemovePreset(alias, name string) error
	RenamePreset(alias, oldName, newName string) error
	RenameModel(ref, name, alias string) error
	SetModelCategory(ref string, category []string) error
	CreateCategory(path []string) error
	RenameCategory(path []string, newName string) error
	DeleteCategory(path []string) error
}

// Addr returns the loopback control address. The host is fixed to 127.0.0.1;
// the port is configurable via LLAMAHERD_API_PORT (default 8517, since
// llama-server itself owns 8080).
func Addr() string { return settings.APIAddr() }

// Serve (blocking) starts the API. Locked to loopback.
func Serve(c Control) error {
	if settings.APIToken() != "" {
		log.Printf("ctl: api listening on http://%s (bearer token required)", Addr())
	} else {
		log.Printf("ctl: api listening on http://%s", Addr())
	}
	var e http.Server
	e.Addr = Addr()
	e.Handler = Handler(c)
	return e.ListenAndServe()
}

// Handler returns the API's HTTP handler (auth included) for c; Serve wraps it
// in a loopback listener, tests mount it on httptest.
func Handler(c Control) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		b, err := c.SnapshotJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})
	mux.HandleFunc("GET /models", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"models": c.Models()})
	})
	mux.HandleFunc("GET /catalog", func(w http.ResponseWriter, r *http.Request) {
		b, err := c.CatalogJSON(r.URL.Query().Get("all") == "1")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})
	mux.HandleFunc("GET /download", func(w http.ResponseWriter, r *http.Request) {
		b, err := c.DownloadJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})
	mux.HandleFunc("GET /download/files", func(w http.ResponseWriter, r *http.Request) {
		ref := r.URL.Query().Get("ref")
		if ref == "" {
			http.Error(w, "ref required", http.StatusBadRequest)
			return
		}
		b, err := c.RepoFilesJSON(ref)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})
	mux.HandleFunc("POST /download", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Ref    string `json:"ref"`
			Repo   string `json:"repo"`
			File   string `json:"file"`
			Vision string `json:"vision"`
			MTP    string `json:"mtp"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ref := req.Ref
		if ref == "" {
			ref = req.Repo
		}
		if err := c.StartDownload(ref, req.File, req.Vision, req.MTP); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /download/cancel", func(w http.ResponseWriter, r *http.Request) {
		if err := c.CancelDownload(); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := c.SetHFToken(req.Token); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /doctor", func(w http.ResponseWriter, r *http.Request) {
		b, err := c.DoctorJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})
	mux.HandleFunc("GET /presets", func(w http.ResponseWriter, r *http.Request) {
		alias := r.URL.Query().Get("alias")
		if alias == "" {
			http.Error(w, "alias required", http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{"alias": alias, "presets": c.Presets(alias)})
	})
	mux.HandleFunc("POST /open", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Alias  string `json:"alias"`
			Preset string `json:"preset"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := c.Open(req.Alias, req.Preset); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /stop", func(w http.ResponseWriter, r *http.Request) {
		if err := c.Stop(); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /save", func(w http.ResponseWriter, r *http.Request) {
		c.SaveConfig()
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /load", func(w http.ResponseWriter, r *http.Request) {
		c.LoadConfig()
		writeJSON(w, map[string]any{"ok": true})
	})

	// Library editing: model = file path or alias; categories are string paths.
	mux.HandleFunc("POST /model/rename", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Model, Name, Alias string }
		decodeAndRun(w, r, &req, func() error { return c.RenameModel(req.Model, req.Name, req.Alias) })
	})
	mux.HandleFunc("POST /model/category", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model    string   `json:"model"`
			Category []string `json:"category"`
		}
		decodeAndRun(w, r, &req, func() error { return c.SetModelCategory(req.Model, req.Category) })
	})
	mux.HandleFunc("POST /category/create", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path []string `json:"path"`
		}
		decodeAndRun(w, r, &req, func() error { return c.CreateCategory(req.Path) })
	})
	mux.HandleFunc("POST /category/rename", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path []string `json:"path"`
			Name string   `json:"name"`
		}
		decodeAndRun(w, r, &req, func() error { return c.RenameCategory(req.Path, req.Name) })
	})
	mux.HandleFunc("POST /category/delete", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path []string `json:"path"`
		}
		decodeAndRun(w, r, &req, func() error { return c.DeleteCategory(req.Path) })
	})

	// config add|update|remove share one body shape:
	// {"alias":"...", "name":"...", "config":{"basic":{...},"advanced":{...}}}
	// For add/update the "config" MAY be partial: missing keys keep the
	// matching existing preset's values (deep-ish merge per top part).
	mux.HandleFunc("POST /config/add", func(w http.ResponseWriter, r *http.Request) {
		mergePreset(w, r, c, false)
	})
	mux.HandleFunc("POST /config/update", func(w http.ResponseWriter, r *http.Request) {
		mergePreset(w, r, c, true)
	})
	mux.HandleFunc("POST /config/remove", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Alias string `json:"alias"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := c.RemovePreset(req.Alias, req.Name); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /config/rename", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Alias   string `json:"alias"`
			Name    string `json:"name"`
			NewName string `json:"newName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := c.RenamePreset(req.Alias, req.Name, req.NewName); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	return withAuth(mux)
}

// withAuth enforces a bearer token when settings.APIToken() is set, accepting
// "Authorization: Bearer <token>" or a "?token=<token>" query parameter. With
// no token configured the API stays open (backward compatible).
func withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := settings.APIToken()
		if tok == "" ||
			r.Header.Get("Authorization") == "Bearer "+tok ||
			r.URL.Query().Get("token") == tok {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// mergePreset handles add/update: the given config replaces the referenced
// preset wholesale; a nil config copies the preset currently applied.
func mergePreset(w http.ResponseWriter, r *http.Request, c Control, update bool) {
	var req struct {
		Alias  string         `json:"alias"`
		Name   string         `json:"name"`
		Config *schema.Config `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// A config without basic/advanced decodes as zero parts; the caller
	// (CLI) is responsible for seeding complete configs, so pass through.
	if update {
		if err := c.UpdatePreset(req.Alias, req.Name, deref(req.Config)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	} else {
		if err := c.AddPreset(req.Alias, req.Name, deref(req.Config)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true})
}

// decodeAndRun decodes the JSON body into req, runs fn and writes {"ok":...}.
func decodeAndRun(w http.ResponseWriter, r *http.Request, req any, fn func() error) {
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := fn(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func deref(p *schema.Config) schema.Config {
	if p == nil {
		return schema.Config{}
	}
	return *p
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
