package main

import (
	"context"
	"fmt"
	"time"

	"llamaherd/hf"
)

// StartDownload begins downloading a Hugging Face model. ref may be
// "owner/name", an "owner/name/file.gguf" path, or a full hf.co URL; file is
// optional when ref already includes it. visionFile optionally pins the mmproj
// sidecar and mtpFile the MTP draft sidecar; for both, "" means "use the
// repository's one when present" and "none" skips it. The download runs in the
// background and reports via Snapshot.Download / GET /download. When another
// transfer is already active the new job is queued and started afterwards.
func (a *App) StartDownload(ref, file, visionFile, mtpFile string) error {
	repo, embedded, err := hf.ParseRef(ref)
	if err != nil {
		return err
	}
	if file == "" {
		file = embedded
	}
	if file != "" {
		if file, err = hf.ValidateFileRef(file); err != nil {
			return err
		}
	}
	a.dlMu.Lock()
	a.dlSeq++
	job := QueuedDownload{
		ID: fmt.Sprintf("dl-%d", a.dlSeq), Repo: repo, File: file,
		Vision: visionFile, MTP: mtpFile, Status: "queued",
	}
	a.dl.Queue = append(a.dl.Queue, job)
	a.dl.Updated = time.Now().Format(time.RFC3339)
	a.dlMu.Unlock()

	a.Log(fmt.Sprintf("hf: queued %s %s", repo, file))
	a.emitState()
	go a.pumpDownloads()
	return nil
}

// PauseDownload cancels the active transfer (its .part is kept for resume) and
// stops the queue from advancing. Queued jobs stay queued.
func (a *App) PauseDownload() error {
	a.dlMu.Lock()
	if a.dl.Active || a.dl.Status == "paused" || len(a.dl.Queue) > 0 {
		a.dl.Paused = true
		a.dlPausing = true
		cancel := a.dlCancel
		a.dlMu.Unlock()
		if cancel != nil {
			cancel()
		}
		a.Log("hf: download paused")
		a.emitState()
		return nil
	}
	a.dlMu.Unlock()
	return fmt.Errorf("no download to pause")
}

// ResumeDownload continues a paused transfer from its .part file and then
// processes the remaining queue.
func (a *App) ResumeDownload() error {
	a.dlMu.Lock()
	if !a.dl.Paused && a.dl.Status != "paused" && len(a.dl.Queue) == 0 {
		a.dlMu.Unlock()
		return fmt.Errorf("no paused download to resume")
	}
	a.dl.Paused = false
	a.dlPausing = false
	if a.dl.Status == "paused" {
		a.dl.Status = "queued"
		a.dl.Message = "resuming"
	}
	a.dl.Updated = time.Now().Format(time.RFC3339)
	a.dlMu.Unlock()
	a.Log("hf: download resumed")
	a.emitState()
	go a.pumpDownloads()
	return nil
}

// RetryDownload re-enqueues the last failed or cancelled job once.
func (a *App) RetryDownload() error {
	a.dlMu.Lock()
	if a.dl.Status != "error" && a.dl.Status != "cancelled" {
		a.dlMu.Unlock()
		return fmt.Errorf("nothing to retry: the last download is %s", a.dl.Status)
	}
	job := a.dlLast
	if job.ID == "" {
		a.dlMu.Unlock()
		return fmt.Errorf("no download to retry")
	}
	if job.Attempts >= 1 {
		a.dlMu.Unlock()
		return fmt.Errorf("retry already used for %s", job.File)
	}
	job.Attempts++
	job.Status = "queued"
	job.Message = "retrying"
	a.dl.Queue = append([]QueuedDownload{job}, a.dl.Queue...)
	a.dl.Paused = false
	a.dlPausing = false
	a.dl.Status = "queued"
	a.dl.Error = ""
	a.dl.Updated = time.Now().Format(time.RFC3339)
	a.dlMu.Unlock()
	a.Log("hf: retrying " + job.File)
	a.emitState()
	go a.pumpDownloads()
	return nil
}

// RemoveQueuedDownload cancels one waiting job. The active transfer is never
// touched.
func (a *App) RemoveQueuedDownload(id string) error {
	a.dlMu.Lock()
	if id == "" {
		a.dlMu.Unlock()
		return fmt.Errorf("download id required")
	}
	if a.dl.Active && a.dl.CurrentID == id {
		a.dlMu.Unlock()
		return fmt.Errorf("download %s is already running; cancel it instead", id)
	}
	for i, q := range a.dl.Queue {
		if q.ID == id {
			a.dl.Queue = append(a.dl.Queue[:i], a.dl.Queue[i+1:]...)
			a.dl.Updated = time.Now().Format(time.RFC3339)
			a.dlMu.Unlock()
			a.Log("hf: removed queued download " + q.File)
			a.emitState()
			return nil
		}
	}
	a.dlMu.Unlock()
	return fmt.Errorf("no queued download with id %s", id)
}

// pumpDownloads runs queued jobs one at a time. Only one pump goroutine exists
// at a time; every exit path clears dlWorker so a later Start/Resume restarts it.
func (a *App) pumpDownloads() {
	a.dlMu.Lock()
	if a.dlWorker {
		a.dlMu.Unlock()
		return
	}
	a.dlWorker = true
	a.dlMu.Unlock()
	defer func() {
		a.dlMu.Lock()
		a.dlWorker = false
		a.dlMu.Unlock()
	}()

	for {
		a.dlMu.Lock()
		if a.dl.Paused || len(a.dl.Queue) == 0 {
			a.dlMu.Unlock()
			return
		}
		job := a.dl.Queue[0]
		a.dl.Queue = a.dl.Queue[1:]
		a.dlLast = job
		ctx, cancel := context.WithCancel(context.Background())
		a.dlCancel = cancel
		a.dl.CurrentID = job.ID
		a.dl.Active = true
		a.dl.Paused = false
		a.dlPausing = false
		a.dl.Repo = job.Repo
		a.dl.File = job.File
		a.dl.Received, a.dl.Total, a.dl.Percent, a.dl.SpeedBps = 0, 0, 0, 0
		a.dl.Status = "downloading"
		a.dl.Error, a.dl.Message = "", ""
		a.dl.Files = []string{}
		a.dl.Updated = time.Now().Format(time.RFC3339)
		a.dlMu.Unlock()
		a.emitState()

		if job.Attempts > 0 {
			time.Sleep(time.Second) // small backoff before the one retry
		}
		a.runDownloadJob(ctx, job, cancel)
	}
}

// runDownloadJob performs one queued download, then records its outcome. The
// queue pump decides whether the next job runs.
func (a *App) runDownloadJob(ctx context.Context, job QueuedDownload, cancel context.CancelFunc) {
	err := a.doDownload(ctx, job.Repo, job.File, job.Vision, job.MTP)
	cancelled := ctx.Err() != nil
	cancel()

	a.dlMu.Lock()
	pausing := a.dlPausing
	a.dlCancel = nil
	a.dlMu.Unlock()

	switch {
	case err == nil:
		a.setDownload(func(d *DownloadState) {
			d.Active = false
			d.Status = "done"
			d.Percent = 100
			d.Message = fmt.Sprintf("installed %s", d.File)
		})
		a.Log("hf: download complete")
		a.notifyEvent("download", "LlamaHerd: download finished", fmt.Sprintf("%s is installed", job.File))
	case pausing:
		a.dlMu.Lock()
		// Put the paused job back at the front so Resume restarts it; the
		// .part file makes the transfer continue where it stopped.
		job.Status = "paused"
		a.dl.Queue = append([]QueuedDownload{job}, a.dl.Queue...)
		a.dl.Active = false
		a.dl.Status = "paused"
		a.dl.Message = "paused; partial file kept for resume"
		a.dl.Updated = time.Now().Format(time.RFC3339)
		a.dlMu.Unlock()
		a.Log("hf: download paused")
	case cancelled:
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
		a.notifyEvent("download", "LlamaHerd: download failed", fmt.Sprintf("%s: %s", job.File, err.Error()))
	}
	a.emitState()
}
