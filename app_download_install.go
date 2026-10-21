package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"llamaherd/control"
	"llamaherd/hf"
	"llamaherd/schema"
	"llamaherd/settings"
	"llamaherd/store"
)

// doDownload resolves the repository, downloads the model and (when present)
// the vision projector, then installs a tuned preset. Progress callbacks
// update the shared state and emit at most ~6 events/second.
func (a *App) doDownload(ctx context.Context, repo, file, visionFile, mtpFile string) error {
	token := settings.HFToken()
	modelsDir := settings.ModelsDir()
	destDir := filepath.Join(modelsDir, repo)
	if !settings.PathInside(modelsDir, destDir) {
		return fmt.Errorf("download destination escapes the models directory")
	}

	info, err := checkDownloadAccess(ctx, repo, token)
	if err != nil {
		return err
	}
	tree, err := hf.Tree(ctx, repo, token)
	if err != nil {
		return downloadAccessError(err, info, token)
	}
	models, err := hf.ResolveModelSet(tree, file)
	if err != nil {
		return err
	}
	shardPaths := make([]string, len(models))
	var setTotal int64
	for i, m := range models {
		shardPaths[i] = m.Path
		setTotal += m.Size
	}
	multi := len(models) > 1
	a.setDownload(func(d *DownloadState) {
		if multi {
			d.Shards = shardPaths
			d.ShardCount = len(models)
		}
	})

	lastEmit := time.Time{}
	completed := int64(0)
	onProgress := func(name string, index int, done, total int64) func(hf.Progress) {
		return func(p hf.Progress) {
			a.setDownload(func(d *DownloadState) {
				d.File = name
				d.ShardIndex = index
				d.Received = done + p.Received
				d.Total = total
				if total > 0 {
					d.Percent = float64(d.Received) / float64(total) * 100
				} else {
					d.Percent = p.Percent
				}
				d.SpeedBps = p.SpeedBps
				d.Status = p.Phase
			})
			if time.Since(lastEmit) > 150*time.Millisecond {
				lastEmit = time.Now()
				a.emitState()
			}
		}
	}

	var modelPath string
	for i, m := range models {
		dest := filepath.Join(destDir, filepath.Base(m.Path))
		if i > 0 && hf.FileMatches(dest, m.Size, m.OID) {
			// A shard that finished before a retry is not fetched again.
			completed += m.Size
			a.setDownload(func(d *DownloadState) { d.Files = append(d.Files, dest) })
			a.Log(fmt.Sprintf("hf: shard %d/%d already complete: %s", i+1, len(models), filepath.Base(m.Path)))
			continue
		}
		p, err := hf.Download(ctx, hf.Options{
			Repo: repo, File: m.Path, DestDir: destDir, Token: token,
			ExpectedSize: m.Size, ExpectedSHA256: m.OID,
			OnProgress: onProgress(m.Path, i+1, completed, setTotal), Retries: 3,
		})
		if err != nil {
			return downloadAccessError(err, info, token)
		}
		modelPath = p
		completed += m.Size
		a.setDownload(func(d *DownloadState) { d.Files = append(d.Files, p) })
	}
	if multi {
		modelPath = filepath.Join(destDir, filepath.Base(models[0].Path))
	}
	if modelPath == "" && len(models) > 0 {
		modelPath = filepath.Join(destDir, filepath.Base(models[0].Path))
	}
	firstPath := models[0].Path

	var mmPath string
	if mm, ok := hf.ResolveMMProj(tree, visionFile); ok && mm.Path != firstPath {
		p, err := hf.Download(ctx, hf.Options{
			Repo: repo, File: mm.Path, DestDir: destDir, Token: token,
			ExpectedSize: mm.Size, ExpectedSHA256: mm.OID,
			OnProgress: onProgress(mm.Path, 0, 0, mm.Size), Retries: 3,
		})
		if err != nil {
			return err
		}
		mmPath = p
		a.setDownload(func(d *DownloadState) { d.Files = append(d.Files, mmPath) })
	}

	var draftPath string
	if draft, ok := hf.ResolveMTPDraft(tree, mtpFile); ok && draft.Path != firstPath {
		p, err := hf.Download(ctx, hf.Options{
			Repo: repo, File: draft.Path, DestDir: destDir, Token: token,
			ExpectedSize: draft.Size, ExpectedSHA256: draft.OID,
			OnProgress: onProgress(draft.Path, 0, 0, draft.Size), Retries: 3,
		})
		if err != nil {
			return err
		}
		draftPath = p
		a.setDownload(func(d *DownloadState) { d.Files = append(d.Files, draftPath) })
	}

	a.installDownloaded(repo, firstPath, modelPath, mmPath, draftPath, provenanceFrom(info))
	return nil
}

// installDownloaded writes a library meta (with the repo/licence recorded at
// download time) and a tuned ~50k-context preset for a freshly downloaded model, then rescans the models directory. Embedded MTP
// heads and downloaded sidecars (vision, MTP draft) are recorded in the meta.
func (a *App) installDownloaded(repo, file, modelPath, mmPath, draftPath string, prov *schema.Provenance) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	id := store.UniqueIDFor(modelPath, a.models)
	meta := store.Meta{
		ID:         id,
		Path:       settings.RelativizeModelPath(modelPath),
		Category:   []string{"Downloaded", repoName(repo)},
		MMProj:     settings.RelativizeModelPath(mmPath),
		Source:     "hf:" + repo + "/" + file,
		Provenance: prov,
	}
	// Trust the GGUF header over a hand-written flag for the embedded MTP head.
	if info, err := store.ReadGGUFInfo(modelPath); err == nil && info.NextNPredictLayers > 0 {
		meta.MTP = true
	}
	if draftPath != "" {
		meta.MTPDraft = settings.RelativizeModelPath(draftPath)
	}
	_ = store.SaveMeta(settings.LibraryDir(), meta)

	if _, err := os.Stat(filepath.Join(settings.ConfigsDir(), id+".json")); err != nil {
		p := a.buildDownloadPreset(id, modelPath, mmPath, draftPath, meta.MTP)
		if err := store.SaveAlias(settings.ConfigsDir(), id, []schema.Preset{p}); err != nil {
			a.Log("hf: could not write config: " + err.Error())
		}
	}
	a.persistModelsDir()
	a.initModels()
	a.initPresets()
	if i := a.indexOfModelPath(modelPath); i >= 0 {
		a.selModel = i
		a.applyPreset(0)
	}
	if draftPath != "" {
		a.Log("hf: MTP draft sidecar installed (" + filepath.Base(draftPath) + "); enable speculative decoding with spec type draft-simple")
	}
	a.Log("hf: installed " + filepath.Base(modelPath) + " (alias " + id + ")")
	a.emitStateLocked()
}

func repoName(repo string) string {
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		return repo[i+1:]
	}
	return repo
}

// buildDownloadPreset builds a hardware-fitted ~50k-context preset for a
// downloaded model (vision sidecar attached when present). An embedded MTP head
// enables speculative decoding (draft-mtp); a separate draft sidecar is
// recorded but left off (FastMTP needs a patched llama.cpp).
func (a *App) buildDownloadPreset(alias, model, mmproj, draft string, embeddedMTP bool) schema.Preset {
	p := schema.Preset{Name: "default", Basic: schema.Basic{
		Model: model, Alias: alias, Host: "127.0.0.1", Port: 8090,
		Context: 51200, GPULayers: "all", ParallelSlots: 1,
		FlashAttention: "on", KCache: "q8_0", VCache: "q8_0",
		Jinja: true, ReasoningFmt: "deepseek",
	}, Advanced: schema.Advanced{Fit: "off", SleepIdle: 300}}
	if mmproj != "" {
		p.Advanced.MMProj = mmproj
		p.Advanced.NoMMProjOffload = true
	}
	switch {
	case embeddedMTP:
		// The model carries NextN/MTP heads: mainline llama.cpp can use them
		// directly via draft-mtp, no separate draft model needed.
		p.Advanced.SpecType = "draft-mtp"
		p.Advanced.DraftNMax = 3
	case draft != "":
		// A separate draft sidecar is available; keep it off by default (some,
		// like FastMTP, need a patched runtime) but pre-fill the path.
		p.Advanced.DraftModel = draft
	}
	return control.FitPreset(p.AsConfig(), 51200).AsPreset("default")
}

// persistModelsDir records the resolved models directory in the user config so
// downloaded models keep being found after a restart.
func (a *App) persistModelsDir() {
	f := settings.Config()
	if strings.TrimSpace(f.ModelsDir) != "" {
		return
	}
	f.ModelsDir = settings.ModelsDir()
	if err := settings.Save(f); err == nil {
		settings.Reload()
	}
}
