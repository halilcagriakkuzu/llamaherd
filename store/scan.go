// Package store is the first phase-2 piece (task 10): real configs/<alias>.json
// persistence and recursive *.gguf model scanning. It deliberately contains NO
// process control — that stays out until task 11 (ServerManager).
package store

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"llamaherd/dummy"
	"llamaherd/settings"
)

// The models directory is configuration, not a constant: callers pass it in
// (the app resolves it via settings.ModelsDir). mmproj*.gguf files are vision
// sidecars — listed, but never loadable models.

// AliasFile is the on-disk shape of configs/<alias>.json: the alias plus its
// named presets, each preset = full basic+advanced field set (the same shape
// the dummy phase used in memory).
type AliasFile struct {
	Alias   string         `json:"alias"`
	Presets []dummy.Preset `json:"presets"`
}

func aliasPath(dir, alias string) string {
	// Aliases are file-safe by construction (lower-cased file-derived names);
	// guard against path tricks anyway.
	safe := filepath.Base(alias + ".json")
	return filepath.Join(dir, safe)
}

// LoadAlias reads configs/<alias>.json. Returns an error the caller can show
// if the file is missing or malformed.
func LoadAlias(dir string, alias string) (AliasFile, error) {
	data, err := os.ReadFile(aliasPath(dir, alias))
	if err != nil {
		return AliasFile{}, err
	}
	var f AliasFile
	if err := json.Unmarshal(data, &f); err != nil {
		return AliasFile{}, fmt.Errorf("parse %s: %w", aliasPath(dir, alias), err)
	}
	if f.Alias == "" {
		f.Alias = alias
	}
	if len(f.Presets) == 0 {
		return AliasFile{}, fmt.Errorf("%s: no presets", aliasPath(dir, alias))
	}
	f.Presets = ResolvePresetPaths(f.Presets)
	return f, nil
}

// ResolvePresetPaths returns a copy of ps with model / mmproj paths resolved to
// absolute paths (shipped configs store them relative to the models directory).
func ResolvePresetPaths(ps []dummy.Preset) []dummy.Preset {
	out := make([]dummy.Preset, len(ps))
	copy(out, ps)
	for i := range out {
		out[i].Basic.Model = settings.ResolveModelPath(out[i].Basic.Model)
		out[i].Advanced.MMProj = settings.ResolveModelPath(out[i].Advanced.MMProj)
		out[i].Advanced.DraftModel = settings.ResolveModelPath(out[i].Advanced.DraftModel)
	}
	return out
}

// portablePresetPaths returns a copy of ps with model / mmproj paths made
// relative to the models directory when possible, so saved configs stay
// machine-independent.
func portablePresetPaths(ps []dummy.Preset) []dummy.Preset {
	out := make([]dummy.Preset, len(ps))
	copy(out, ps)
	for i := range out {
		out[i].Basic.Model = settings.RelativizeModelPath(out[i].Basic.Model)
		out[i].Advanced.MMProj = settings.RelativizeModelPath(out[i].Advanced.MMProj)
		out[i].Advanced.DraftModel = settings.RelativizeModelPath(out[i].Advanced.DraftModel)
	}
	return out
}

// SaveAlias writes configs/<alias>.json. os.MkdirAll keeps it working from
// any cwd relative base; the app passes "configs".
func SaveAlias(dir string, alias string, presets []dummy.Preset) error {
	if len(presets) == 0 {
		return fmt.Errorf("refusing to save empty preset list for %s", alias)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(AliasFile{Alias: alias, Presets: portablePresetPaths(presets)}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(aliasPath(dir, alias), append(data, '\n'), 0o644)
}

// ScanModels walks root recursively and returns one entry per loadable *.gguf.
// mmproj*.gguf vision sidecars are NOT listed: each is attached to the loadable
// models in the same directory (Model.MMProj). Alias is left empty — the caller
// resolves aliases (it knows the built-in preset table).
func ScanModels(root string) []dummy.Model {
	var loads []string
	mmByDir := map[string][]string{}

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil // unreadable subtree: skip silently, scan what we can
		}
		name := d.Name()
		if !strings.EqualFold(filepath.Ext(name), ".gguf") {
			return nil
		}
		if isMTPSidecar(name) {
			return nil // MTP draft sidecar, recorded in meta.mtpDraft; not a model
		}
		if strings.Contains(strings.ToLower(name), "mmproj") {
			dir := filepath.Dir(path)
			mmByDir[dir] = append(mmByDir[dir], path)
		} else {
			loads = append(loads, path)
		}
		return nil
	})

	sort.Strings(loads)
	models := make([]dummy.Model, 0, len(loads))
	for _, l := range loads {
		m := dummy.Model{Display: display(strings.TrimSuffix(filepath.Base(l), filepath.Ext(l))), Path: l}
		if mms := mmByDir[filepath.Dir(l)]; len(mms) > 0 {
			sort.Strings(mms)
			m.MMProj = mms[0]
		}
		models = append(models, m)
	}
	return models
}

// DerivedAlias turns a model file base name into a config-file-safe alias
// (the same transformation when no preset table entry claims the path).
func DerivedAlias(modelPath string) string {
	base := strings.TrimSuffix(filepath.Base(modelPath), filepath.Ext(modelPath))
	lower := strings.ToLower(base)
	return strings.ReplaceAll(lower, ".", "-")
}

// isMTPSidecar reports whether a file name looks like a separate MTP draft
// sidecar (e.g. HauhauCS "fastmtp*.gguf"). These are draft models for
// --spec-draft-model, never loadable target models.
func isMTPSidecar(name string) bool {
	b := strings.ToLower(name)
	for _, frag := range []string{"fastmtp", "mtp-draft", "mtp_draft", "draft-mtp", "draft_mtp"} {
		if strings.Contains(b, frag) {
			return true
		}
	}
	return false
}

func display(base string) string {
	if len(base) <= 40 {
		return base
	}
	return base[:16] + "-...-" + base[len(base)-18:]
}

// elide derives a list-display name from a file base name.
func elide(s string) string { return display(s) }
