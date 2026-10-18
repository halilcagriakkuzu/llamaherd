package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"llamaherd/control"
	"llamaherd/hf"
	"llamaherd/settings"
)

// DownloadState is the live state of the model downloader, surfaced in the
// Snapshot (GUI) and via GET /download (CLI/API).
type DownloadState struct {
	Active   bool     `json:"active"`
	Repo     string   `json:"repo"`
	File     string   `json:"file"`
	Dest     string   `json:"dest"`
	Received int64    `json:"received"`
	Total    int64    `json:"total"`
	Percent  float64  `json:"percent"`
	SpeedBps float64  `json:"speedBps"`
	Status   string   `json:"status"` // idle|downloading|verifying|done|error|cancelled
	Error    string   `json:"error"`
	Message  string   `json:"message"`
	Files    []string `json:"files"`
	Updated  string   `json:"updated"`
}

// DownloadSnapshot returns the cached downloader state (cheap; snapshot-safe).
func (a *App) DownloadSnapshot() DownloadState {
	a.dlMu.Lock()
	defer a.dlMu.Unlock()
	return a.dl
}

// setDownload mutates the downloader state under the lock.
func (a *App) setDownload(fn func(*DownloadState)) DownloadState {
	a.dlMu.Lock()
	defer a.dlMu.Unlock()
	fn(&a.dl)
	a.dl.Updated = time.Now().Format(time.RFC3339)
	return a.dl
}

// DownloadJSON marshals the downloader state (control API GET /download).
func (a *App) DownloadJSON() ([]byte, error) {
	return json.Marshal(a.DownloadSnapshot())
}

// CatalogJSON returns the curated catalog scored against this machine
// (control API GET /catalog).
func (a *App) CatalogJSON() ([]byte, error) {
	return json.Marshal(catalogView())
}

// GetCatalog returns the curated catalog scored against this machine (Wails
// binding for the download panel).
func (a *App) GetCatalog() map[string]any {
	return catalogView()
}

// GetHFTokenStatus reports whether a token is configured (Wails binding).
func (a *App) GetHFTokenStatus() map[string]any {
	return a.HFTokenStatus()
}

// catalogView is the recommendation table for the current hardware.
func catalogView() map[string]any {
	vram := control.TotalVramMB()
	ram := totalRAMMB()
	return map[string]any{
		"vramMB":  vram,
		"ramMB":   ram,
		"entries": hf.Recommend(vram, ram),
		"starter": hf.Starter(),
	}
}

// SetHFToken stores (or clears) the Hugging Face token safely (0600 file).
func (a *App) SetHFToken(token string) error {
	if err := settings.SaveHFToken(token); err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		a.Log("hf: token cleared")
	} else {
		a.Log("hf: token saved to " + settings.HFTokenFile())
	}
	a.emitState()
	return nil
}

// HFTokenStatus reports whether a token is configured and where from.
func (a *App) HFTokenStatus() map[string]any {
	src := settings.HFTokenSource()
	return map[string]any{
		"set":    src != "",
		"source": src,
		"path":   settings.HFTokenFile(),
	}
}

// CancelDownload stops an in-flight download (the .part is kept for resume).
func (a *App) CancelDownload() error {
	a.dlMu.Lock()
	cancel := a.dlCancel
	a.dlMu.Unlock()
	if cancel == nil {
		return fmt.Errorf("no download in progress")
	}
	cancel()
	return nil
}

// totalRAMMB is the cross-platform total system RAM (control.TotalRAMMB).
func totalRAMMB() int { return control.TotalRAMMB() }
