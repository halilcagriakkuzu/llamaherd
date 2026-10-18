// library.go — per-model metadata (library/<id>.json): where a model sits in
// the category tree (family > series > group), its paired mmproj sidecar and
// where it came from. The weights directory stays clean; this is the only
// place categories live.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"llamaherd/schema"
	"llamaherd/settings"
)

// LibraryDir holds one meta file per model.
const LibraryDir = "library"

// Meta is the on-disk shape of library/<id>.json. An empty Category means
// "not placed yet" (shown as Uncategorized).
type Meta struct {
	ID       string   `json:"id"`
	Path     string   `json:"path"`
	Category []string `json:"category"`
	MMProj   string   `json:"mmproj,omitempty"`
	MTP      bool     `json:"mtp,omitempty"`
	// MTPDraft is an optional separate MTP draft sidecar (e.g. HauhauCS
	// FastMTP). It is used with a draft-* spec type via advanced.draftModel
	// (--spec-draft-model); FastMTP itself needs a patched llama.cpp, so the
	// downloaded preset leaves it disabled unless the user enables it.
	MTPDraft string `json:"mtpDraft,omitempty"`
	Quant    string `json:"quant,omitempty"`
	Source   string `json:"source,omitempty"`
}

func metaPath(dir, id string) string {
	return filepath.Join(dir, filepath.Base(id+".json"))
}

// LoadMeta reads library/<id>.json; ok=false if absent or malformed.
func LoadMeta(dir, id string) (Meta, bool) {
	data, err := os.ReadFile(metaPath(dir, id))
	if err != nil {
		return Meta{}, false
	}
	var m Meta
	if json.Unmarshal(data, &m) != nil {
		return Meta{}, false
	}
	return m, true
}

// SaveMeta writes library/<m.ID>.json.
func SaveMeta(dir string, m Meta) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath(dir, m.ID), append(data, '\n'), 0o644)
}

// ApplyLibrary merges meta files into the scanned models. A model without a
// meta file gets a stub written (Category empty) so it shows up as
// Uncategorized and can be placed later. Returns how many stubs were created.
func ApplyLibrary(dir string, models []schema.Model) (created int) {
	for i := range models {
		m := &models[i]
		id := DerivedAlias(m.Path)
		meta, ok := LoadMeta(dir, id)
		if !ok {
			meta = Meta{ID: id, Path: settings.RelativizeModelPath(m.Path), MMProj: settings.RelativizeModelPath(m.MMProj)}
			if SaveMeta(dir, meta) == nil {
				created++
			}
		}
		m.Category = meta.Category
		// Embedded MTP head: trust the GGUF header over a hand-written flag.
		m.MTP = meta.MTP
		if info, err := ReadGGUFInfo(m.Path); err == nil && info.NextNPredictLayers > 0 {
			m.MTP = true
		}
		m.Source = meta.Source
		if meta.MMProj != "" {
			p := settings.ResolveModelPath(meta.MMProj)
			if _, err := os.Stat(p); err == nil {
				m.MMProj = p // explicit override wins
			}
		}
		if meta.MTPDraft != "" {
			if p := settings.ResolveModelPath(meta.MTPDraft); p != "" {
				if _, err := os.Stat(p); err == nil {
					m.MTPDraft = p
				}
			}
		}
	}
	return created
}
