package main

import (
	"context"
	"fmt"

	"llamaherd/hf"
)

// StartDownload begins downloading a Hugging Face model. ref may be
// "owner/name", an "owner/name/file.gguf" path, or a full hf.co URL; file is
// optional when ref already includes it. visionFile optionally pins the mmproj
// sidecar; otherwise the repository's projector is used when present. The
// download runs in the background and reports via Snapshot.Download /
// GET /download.
func (a *App) StartDownload(ref, file, visionFile string) error {
	repo, embedded := hf.ParseRef(ref)
	if file == "" {
		file = embedded
	}
	if repo == "" {
		return fmt.Errorf("a Hugging Face repo (owner/name) is required")
	}
	a.dlMu.Lock()
	if a.dl.Active {
		cur := a.dl
		a.dlMu.Unlock()
		return fmt.Errorf("a download is already in progress: %s/%s", cur.Repo, cur.File)
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.dlCancel = cancel
	a.dlMu.Unlock()

	a.setDownload(func(d *DownloadState) {
		*d = DownloadState{Active: true, Repo: repo, File: file, Status: "downloading", Files: []string{}}
	})
	a.Log(fmt.Sprintf("hf: downloading %s %s", repo, file))
	go a.runDownloadJob(ctx, repo, file, visionFile)
	a.emitState()
	return nil
}

// runDownloadJob performs the download, then finalises the shared state.
func (a *App) runDownloadJob(ctx context.Context, repo, file, visionFile string) {
	defer func() {
		a.dlMu.Lock()
		a.dlCancel = nil
		a.dlMu.Unlock()
	}()
	err := a.doDownload(ctx, repo, file, visionFile)
	switch {
	case err == nil:
		a.setDownload(func(d *DownloadState) {
			d.Active = false
			d.Status = "done"
			d.Percent = 100
			d.Message = fmt.Sprintf("installed %s", d.File)
		})
		a.Log("hf: download complete")
	case ctx.Err() != nil:
		a.setDownload(func(d *DownloadState) {
			d.Active = false
			d.Status = "cancelled"
			d.Message = "cancelled; partial file kept for resume"
		})
		a.Log("hf: download cancelled")
	default:
		a.setDownload(func(d *DownloadState) {
			d.Active = false
			d.Status = "error"
			d.Error = err.Error()
		})
		a.Log("hf: download failed: " + err.Error())
	}
	a.emitState()
}
