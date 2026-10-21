package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"llamaherd/control"
	"llamaherd/doctor"
	"llamaherd/hf"
	"llamaherd/settings"
)

// QueuedDownload is one download job waiting behind the active transfer.
// Vision/MTP carry the optional sidecar choices made in the download dialog.
type QueuedDownload struct {
	ID       string `json:"id"`
	Repo     string `json:"repo"`
	File     string `json:"file"`
	Vision   string `json:"vision,omitempty"`
	MTP      string `json:"mtp,omitempty"`
	Status   string `json:"status"` // queued|paused
	Message  string `json:"message,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
}

// DownloadState is the live state of the model downloader, surfaced in the
// Snapshot (GUI) and via GET /download (CLI/API).
type DownloadState struct {
	Active    bool             `json:"active"`
	CurrentID string           `json:"currentId"`
	Paused    bool             `json:"paused"`
	Repo      string           `json:"repo"`
	File      string           `json:"file"`
	Dest      string           `json:"dest"`
	Received  int64            `json:"received"`
	Total     int64            `json:"total"`
	Percent   float64          `json:"percent"`
	SpeedBps  float64          `json:"speedBps"`
	Status    string           `json:"status"` // idle|queued|downloading|verifying|paused|done|error|cancelled
	Error     string           `json:"error"`
	Message   string           `json:"message"`
	Files     []string         `json:"files"`
	Queue     []QueuedDownload `json:"queue"`
	Updated   string           `json:"updated"`
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
func (a *App) CatalogJSON(all bool) ([]byte, error) {
	if all {
		return json.Marshal(fullCatalogView())
	}
	return json.Marshal(catalogView())
}

// GetCatalog returns the curated catalog scored against this machine (Wails
// binding for the download panel).
func (a *App) GetCatalog() map[string]any {
	return catalogView()
}

// GetFullCatalog returns every catalog entry scored for this machine, including
// the ones that do not fit (Wails binding for the "browse all" screen).
func (a *App) GetFullCatalog() map[string]any {
	return fullCatalogView()
}

var catalogRefreshAt time.Time

// RefreshCatalog fetches the newest model catalog from the repository's main
// branch (at most every 10 minutes) and reports the active catalog's source.
// Network errors are not fatal: the embedded/cached catalog stays in use.
func (a *App) RefreshCatalog() map[string]any {
	if time.Since(catalogRefreshAt) > 10*time.Minute {
		catalogRefreshAt = time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if changed, err := hf.RefreshCatalog(ctx); err != nil {
			a.Log("catalog: refresh failed (using " + hf.Info().Source + " copy): " + err.Error())
		} else if changed {
			a.Log(fmt.Sprintf("catalog: updated to %s (%d models)", hf.Info().Updated, hf.Info().Count))
		}
	}
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
	recs := hf.Recommend(vram, ram)
	free, total, _ := doctor.DiskFree(settings.ModelsDir())
	return map[string]any{
		"vramMB":    vram,
		"ramMB":     ram,
		"picks":     hf.Picks(recs),
		"fitting":   len(recs),        // models that suit this machine
		"total":     hf.FamilyCount(), // models in the catalog
		"info":      hf.Info(),        // where the catalog came from
		"starter":   hf.Starter(),
		"diskFree":  free,
		"diskTotal": total,
		"modelsDir": settings.ModelsDir(),
	}
}

// fullCatalogView lists every catalog entry, fitting or not.
func fullCatalogView() map[string]any {
	vram := control.TotalVramMB()
	ram := totalRAMMB()
	return map[string]any{
		"vramMB":  vram,
		"ramMB":   ram,
		"entries": hf.Browse(vram, ram),
		"info":    hf.Info(),
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
// When the downloader is paused, the paused job is dropped instead.
func (a *App) CancelDownload() error {
	a.dlMu.Lock()
	cancel := a.dlCancel
	if cancel != nil {
		a.dlPausing = false
		a.dlMu.Unlock()
		cancel()
		return nil
	}
	if a.dl.Status == "paused" && len(a.dl.Queue) > 0 {
		a.dl.Queue = a.dl.Queue[1:]
		a.dl.Status = "cancelled"
		a.dl.Paused = false
		a.dl.Message = "cancelled; partial file kept for resume"
		a.dl.Updated = time.Now().Format(time.RFC3339)
		a.dlMu.Unlock()
		a.Log("hf: download cancelled")
		a.emitState()
		return nil
	}
	a.dlMu.Unlock()
	return fmt.Errorf("no download in progress")
}

// totalRAMMB is the cross-platform total system RAM (control.TotalRAMMB).
func totalRAMMB() int { return control.TotalRAMMB() }

// ListRepoFiles lists the .gguf files of a Hugging Face repository (models,
// vision projectors, MTP drafts) so the UI can offer them as a choice. ref is
// "owner/name" or an hf.co URL.
func (a *App) ListRepoFiles(ref string) (hf.RepoListing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return hf.ListRepo(ctx, ref, hf.Token())
}

// RepoFilesJSON is ListRepoFiles for the control API (GET /download/files).
func (a *App) RepoFilesJSON(ref string) ([]byte, error) {
	l, err := a.ListRepoFiles(ref)
	if err != nil {
		return nil, err
	}
	return json.Marshal(l)
}
