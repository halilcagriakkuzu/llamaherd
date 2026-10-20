package main

// app_import.go — "Import model…": bring local .gguf files (and an optional
// vision projector) into the models directory, create library metadata and
// fitted presets. Supports batches, duplicate detection and one common mmproj.

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"llamaherd/hf"
	"llamaherd/schema"
	"llamaherd/settings"
	"llamaherd/store"
)

// ImportModel is the single-file compatibility wrapper around ImportModels.
func (a *App) ImportModel(srcPath, mmprojPath string, move bool) error {
	rep, err := a.ImportModels([]string{srcPath}, mmprojPath, move, false)
	if err != nil {
		return err
	}
	if len(rep.Items) == 0 {
		return fmt.Errorf("nothing to import")
	}
	it := rep.Items[0]
	if it.Status == "imported" {
		return nil
	}
	if it.Message != "" {
		return errors.New(it.Message)
	}
	return fmt.Errorf("import failed")
}

// ImportModels imports every file in paths (files or directories). Duplicate
// files are skipped unless allowDuplicate is true; each file gets its own
// report item. One mmprojPath may be attached to the whole batch.
func (a *App) ImportModels(paths []string, mmprojPath string, move, allowDuplicate bool) (schema.ImportReport, error) {
	if !a.importMu.TryLock() {
		return schema.ImportReport{}, fmt.Errorf("another import is already running")
	}
	defer a.importMu.Unlock()

	var report schema.ImportReport
	srcs, problems := expandImportPaths(paths)
	report.Items = append(report.Items, problems...)
	if len(srcs) == 0 && len(problems) == 0 {
		return report, fmt.Errorf("choose at least one .gguf file or folder")
	}

	var mmSrc string
	var err error
	if strings.TrimSpace(mmprojPath) != "" && !strings.EqualFold(strings.TrimSpace(mmprojPath), "none") {
		if mmSrc, err = cleanGGUF(mmprojPath, "mmproj"); err != nil {
			return report, err
		}
	}

	modelsDir := settings.ModelsDir()
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		return report, fmt.Errorf("create models directory: %w", err)
	}

	var mmDest string
	var imported []string
	for _, src := range srcs {
		item := schema.ImportItem{Source: src}
		if store.IsSidecarFile(src) {
			item.Status, item.Message = "error", "this looks like a support file (mmproj/MTP), not a loadable model"
			report.Items = append(report.Items, item)
			continue
		}
		if src == mmSrc {
			item.Status, item.Message = "error", "the model and the vision projector are the same file"
			report.Items = append(report.Items, item)
			continue
		}
		dest, status, msg := a.importOne(src, mmSrc, &mmDest, move, allowDuplicate, modelsDir)
		item.Dest, item.Status, item.Message = dest, status, msg
		if status == "imported" {
			imported = append(imported, dest)
		}
		report.Items = append(report.Items, item)
	}

	if len(imported) > 0 {
		aliases := a.installImportedBatch(imported, mmDest)
		for i := range report.Items {
			if report.Items[i].Status == "imported" {
				report.Items[i].Alias = aliases[report.Items[i].Dest]
			}
		}
	}
	return report, nil
}

// CheckImport previews duplicate results before the user commits to a batch.
func (a *App) CheckImport(paths []string) []schema.ImportCheck {
	srcs, problems := expandImportPaths(paths)
	checks := make([]schema.ImportCheck, 0, len(srcs)+len(problems))
	for _, p := range problems {
		checks = append(checks, schema.ImportCheck{Path: p.Source, Error: p.Message})
	}
	for _, src := range srcs {
		if store.IsSidecarFile(src) {
			checks = append(checks, schema.ImportCheck{Path: src, Error: "support file (mmproj/MTP), not a model"})
			continue
		}
		c, err := a.checkImportPath(src)
		if err != nil {
			c.Error = err.Error()
		}
		checks = append(checks, c)
	}
	return checks
}

// checkImportPath finds an existing library model with the same content. Sizes
// are compared first; only size matches pay for the sha256 scan.
func (a *App) checkImportPath(src string) (schema.ImportCheck, error) {
	st, err := os.Stat(src)
	if err != nil {
		return schema.ImportCheck{Path: src}, err
	}
	check := schema.ImportCheck{Path: src, Size: st.Size()}

	type candidate struct {
		path  string
		alias string
		size  int64
	}
	var candidates []candidate
	a.stateMu.RLock()
	for _, m := range a.models {
		if m.IsMMProj {
			continue
		}
		size := m.FileSize
		if size == 0 {
			if s, err := os.Stat(m.Path); err == nil {
				size = s.Size()
			}
		}
		candidates = append(candidates, candidate{path: m.Path, alias: m.Alias, size: size})
	}
	a.stateMu.RUnlock()

	var sameSize []candidate
	for _, c := range candidates {
		if settings.SamePath(c.path, src) {
			check.Duplicate = true
			check.ExistingPath = c.path
			check.ExistingAlias = c.alias
			check.Size = c.size
			return check, nil
		}
		if c.size == st.Size() {
			sameSize = append(sameSize, c)
		}
	}
	if len(sameSize) == 0 {
		return check, nil
	}
	a.Log(fmt.Sprintf("import: checking %d same-size candidate(s) for %s", len(sameSize), filepath.Base(src)))
	srcHash, err := hf.FileSHA256(src)
	if err != nil {
		return check, err
	}
	for _, c := range sameSize {
		if h, err := hf.FileSHA256(c.path); err == nil && h == srcHash {
			check.Duplicate = true
			check.ExistingPath = c.path
			check.ExistingAlias = c.alias
			return check, nil
		}
	}
	return check, nil
}

// importOne copies/moves one file and (when needed) the vision projector.
func (a *App) importOne(src, mmSrc string, mmDest *string, move, allowDuplicate bool, modelsDir string) (dest, status, msg string) {
	check, err := a.checkImportPath(src)
	if err != nil {
		return "", "error", err.Error()
	}
	if check.Duplicate && !allowDuplicate {
		return "", "duplicate", fmt.Sprintf("already exists as %s (%s)", check.ExistingAlias, check.ExistingPath)
	}

	inModels := settings.PathInside(modelsDir, src)
	if inModels {
		dest = src
	} else {
		dest = filepath.Join(modelsDir, filepath.Base(src))
		if _, err := os.Stat(dest); err == nil {
			return "", "error", fmt.Sprintf("a file named %q already exists in the models directory", filepath.Base(src))
		}
		a.Log("import: copying " + src + " → " + dest)
		if err := transferFile(src, dest, move); err != nil {
			return "", "error", fmt.Sprintf("import %s: %v", filepath.Base(src), err)
		}
	}

	if mmSrc != "" && *mmDest == "" {
		if settings.PathInside(modelsDir, mmSrc) {
			*mmDest = mmSrc
		} else {
			md := filepath.Join(modelsDir, filepath.Base(mmSrc))
			if _, err := os.Stat(md); err == nil {
				if !inModels {
					_ = os.Remove(dest)
				}
				return "", "error", fmt.Sprintf("a file named %q already exists in the models directory", filepath.Base(mmSrc))
			}
			a.Log("import: copying vision projector " + mmSrc + " → " + md)
			if err := transferFile(mmSrc, md, move); err != nil {
				if !inModels {
					_ = os.Remove(dest)
				}
				return "", "error", fmt.Sprintf("import %s: %v", filepath.Base(mmSrc), err)
			}
			*mmDest = md
		}
	}
	return dest, "imported", ""
}

// expandImportPaths turns files and directories into a deduplicated file list.
func expandImportPaths(paths []string) ([]string, []schema.ImportItem) {
	var files []string
	var problems []schema.ImportItem
	seen := map[string]bool{}
	add := func(p string) {
		p = filepath.Clean(p)
		if !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}
	for _, raw := range paths {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(settings.ResolveModelPath(p))
		if err != nil {
			problems = append(problems, schema.ImportItem{Source: p, Status: "error", Message: err.Error()})
			continue
		}
		st, err := os.Stat(abs)
		if err != nil {
			problems = append(problems, schema.ImportItem{Source: abs, Status: "error", Message: err.Error()})
			continue
		}
		if !st.IsDir() {
			if !strings.EqualFold(filepath.Ext(abs), ".gguf") {
				problems = append(problems, schema.ImportItem{Source: abs, Status: "error", Message: "file must end in .gguf"})
				continue
			}
			add(abs)
			continue
		}
		before := len(files)
		_ = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(path), ".gguf") && !store.IsSidecarFile(path) {
				add(path)
			}
			return nil
		})
		if len(files) == before {
			problems = append(problems, schema.ImportItem{Source: abs, Status: "error", Message: "no .gguf models found in this folder"})
		}
	}
	return files, problems
}

// cleanGGUF validates a user-picked file: it must exist, be a regular file and
// carry a .gguf extension.
func cleanGGUF(path, what string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", fmt.Errorf("choose a %s file", what)
	}
	abs, err := filepath.Abs(settings.ResolveModelPath(p))
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("%s file: %w", what, err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("%s path is a directory, not a .gguf file", what)
	}
	if !strings.EqualFold(filepath.Ext(abs), ".gguf") {
		return "", fmt.Errorf("%s file must end in .gguf", what)
	}
	return abs, nil
}

// transferFile copies src to dest (creating them), then removes src when move
// is true. A same-filesystem move is a rename, which is instant and free.
func transferFile(src, dest string, move bool) error {
	if move {
		if err := os.Rename(src, dest); err == nil {
			return nil
		}
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dest)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dest)
		return err
	}
	if move {
		return os.Remove(src)
	}
	return nil
}

// installImportedBatch writes library metadata and fitted presets for the
// imported models, then rescans once. Returns dest → alias.
func (a *App) installImportedBatch(dests []string, mmprojPath string) map[string]string {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	aliases := make(map[string]string, len(dests))
	for _, dest := range dests {
		id := store.UniqueIDFor(dest, a.models)
		meta := store.Meta{
			ID:     id,
			Path:   settings.RelativizeModelPath(dest),
			MMProj: settings.RelativizeModelPath(mmprojPath),
			Source: "local",
		}
		if info, err := store.ReadGGUFInfo(dest); err == nil {
			if info.NextNPredictLayers > 0 {
				meta.MTP = true
			}
			meta.Quant = store.QuantFromName(dest)
		}
		if err := store.SaveMeta(settings.LibraryDir(), meta); err != nil {
			a.Log("import: metadata FAILED for " + filepath.Base(dest) + ": " + err.Error())
			continue
		}
		if _, err := os.Stat(filepath.Join(settings.ConfigsDir(), id+".json")); err != nil {
			p := a.buildDownloadPreset(id, dest, mmprojPath, "", meta.MTP)
			if err := store.SaveAlias(settings.ConfigsDir(), id, []schema.Preset{p}); err != nil {
				a.Log("import: config FAILED for " + id + ": " + err.Error())
			}
		}
		aliases[dest] = id
	}
	a.persistModelsDir()
	a.initModels()
	a.initPresets()
	if len(dests) > 0 {
		if i := a.indexOfModelPath(dests[len(dests)-1]); i >= 0 {
			a.selModel = i
			a.applyPreset(0)
		}
	}
	a.reloadBenchmarksLocked()
	a.dirty = false
	a.Log(fmt.Sprintf("import: installed %d model(s)", len(dests)))
	if mmprojPath != "" {
		a.Log("import: vision projector attached (" + filepath.Base(mmprojPath) + ")")
	}
	a.emitStateLocked()
	return aliases
}
