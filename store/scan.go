// Package store persists configs/<alias>.json files and scans the models
// directory recursively for *.gguf files. It deliberately contains NO process
// control; that lives in the control package.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"llamaherd/schema"
	"llamaherd/settings"
)

// The models directory is configuration, not a constant: callers pass it in
// (the app resolves it via settings.ModelsDir). mmproj*.gguf files are vision
// sidecars — listed, but never loadable models.

// AliasFile is the on-disk shape of configs/<alias>.json: the alias plus its
// named presets, each preset = full basic+advanced field set (the same shape
// the in-memory form).
type AliasFile struct {
	Alias   string          `json:"alias"`
	Presets []schema.Preset `json:"presets"`
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
func ResolvePresetPaths(ps []schema.Preset) []schema.Preset {
	out := make([]schema.Preset, len(ps))
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
func portablePresetPaths(ps []schema.Preset) []schema.Preset {
	out := make([]schema.Preset, len(ps))
	copy(out, ps)
	for i := range out {
		out[i].Basic.Model = settings.RelativizeModelPath(out[i].Basic.Model)
		out[i].Advanced.MMProj = settings.RelativizeModelPath(out[i].Advanced.MMProj)
		out[i].Advanced.DraftModel = settings.RelativizeModelPath(out[i].Advanced.DraftModel)
	}
	return out
}

// SaveAlias writes configs/<alias>.json, creating dir if needed.
func SaveAlias(dir string, alias string, presets []schema.Preset) error {
	if len(presets) == 0 {
		return fmt.Errorf("refusing to save empty preset list for %s", alias)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := AliasJSON(alias, presets)
	if err != nil {
		return err
	}
	return os.WriteFile(aliasPath(dir, alias), append(data, '\n'), 0o644)
}

// AliasJSON renders the portable (models-dir-relative) preset file for export
// and for SaveAlias.
func AliasJSON(alias string, presets []schema.Preset) ([]byte, error) {
	return json.MarshalIndent(AliasFile{Alias: alias, Presets: portablePresetPaths(presets)}, "", "  ")
}

// ScanModels walks root recursively and returns one entry per loadable *.gguf.
// mmproj*.gguf vision sidecars are NOT listed: each is attached to the loadable
// models in the same directory (Model.MMProj). Alias is left empty — the caller
// resolves aliases.
func ScanModels(root string) []schema.Model {
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
	models := make([]schema.Model, 0, len(loads))
	for _, l := range loads {
		m := schema.Model{Display: display(strings.TrimSuffix(filepath.Base(l), filepath.Ext(l))), Path: l}
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

// UniqueIDs returns one stable library/config id per model. Non-colliding files
// keep the historical file-derived alias. When two files share a base name the
// lexicographically first path keeps it and the rest get a short path hash;
// this prevents two models from silently sharing configs/<alias>.json.
func UniqueIDs(models []schema.Model) []string {
	groups := map[string][]int{}
	for i, m := range models {
		base := DerivedAlias(m.Path)
		groups[base] = append(groups[base], i)
	}
	ids := make([]string, len(models))
	for base, idxs := range groups {
		paths := append([]int(nil), idxs...)
		sort.Slice(paths, func(i, j int) bool { return models[paths[i]].Path < models[paths[j]].Path })
		for rank, i := range paths {
			if rank == 0 {
				ids[i] = base
				continue
			}
			ids[i] = base + "-" + shortPathHash(filepath.Dir(models[i].Path))
		}
	}
	return ids
}

// UniqueIDFor returns the id a new model at path will have once it is part of
// others. It lets the downloader persist the same id the next scan will assign.
func UniqueIDFor(path string, others []schema.Model) string {
	models := make([]schema.Model, 0, len(others)+1)
	found := false
	for _, m := range others {
		if m.Path == path {
			found = true
		}
		models = append(models, m)
	}
	if !found {
		models = append(models, schema.Model{Path: path})
	}
	ids := UniqueIDs(models)
	for i, m := range models {
		if m.Path == path {
			return ids[i]
		}
	}
	return DerivedAlias(path)
}

// shortPathHash is a 6-hex-digit, path-stable disambiguator for duplicate base
// names. It only changes when the directory moves.
func shortPathHash(dir string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(dir)))
	return hex.EncodeToString(sum[:3])
}

// IsSidecarFile reports whether a .gguf looks like a support file (mmproj or an
// MTP draft) rather than a loadable model.
func IsSidecarFile(path string) bool {
	b := strings.ToLower(filepath.Base(path))
	return strings.Contains(b, "mmproj") || isMTPSidecar(b)
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

// DisplayName is the file-derived list name for a model path.
func DisplayName(modelPath string) string {
	return display(strings.TrimSuffix(filepath.Base(modelPath), filepath.Ext(modelPath)))
}

func display(base string) string {
	if len(base) <= 40 {
		return base
	}
	return base[:16] + "-...-" + base[len(base)-18:]
}

// quantRe recognises the common GGUF quant labels in a file name.
var quantRe = regexp.MustCompile(`(?i)(IQ\d+_[A-Z0-9_]+|Q\d+_K_[A-Z]+|Q\d+_\d+|Q\d+_K|Q\d+_0|BF16|F32|F16)`)

// QuantFromName extracts a quant label (Q4_K_M, UD-IQ4_XS, Q8_0, ...) from a
// model file name, or "" when none is recognisable.
func QuantFromName(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ReplaceAll(base, "-", "_")
	if m := quantRe.FindStringSubmatch(base); m != nil {
		return strings.ToUpper(m[1])
	}
	return ""
}
