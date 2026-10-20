// library.go — per-model metadata (library/<id>.json): where a model sits in
// the category tree (family > series > group), its paired mmproj sidecar and
// where it came from. The weights directory stays clean; this is the only
// place categories live.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"llamaherd/schema"
	"llamaherd/settings"
)

// LibraryDir holds one meta file per model.
const LibraryDir = "library"

// Meta is the on-disk shape of library/<id>.json. An empty Category means
// "not placed yet" (shown as Uncategorized).
type Meta struct {
	ID   string `json:"id"` // always the file-derived id (stable across renames)
	Path string `json:"path"`
	// Name overrides the display name and Alias the llama-server alias /
	// configs/<alias>.json key; empty means "derived from the file name".
	Name     string   `json:"name,omitempty"`
	Alias    string   `json:"alias,omitempty"`
	Category []string `json:"category"`
	Favorite bool     `json:"favorite,omitempty"`
	// LastUsed is the Unix time (seconds) of the last selection, omitted until
	// the model has been used.
	LastUsed int64  `json:"lastUsed,omitempty"`
	MMProj   string `json:"mmproj,omitempty"`
	MTP      bool   `json:"mtp,omitempty"`
	// MTPDraft is an optional separate MTP draft sidecar (e.g. HauhauCS
	// FastMTP). It is used with a draft-* spec type via advanced.draftModel
	// (--spec-draft-model); FastMTP itself needs a patched llama.cpp, so the
	// downloaded preset leaves it disabled unless the user enables it.
	MTPDraft string `json:"mtpDraft,omitempty"`
	Quant    string `json:"quant,omitempty"`
	Source   string `json:"source,omitempty"`
	// Provenance is the source repo and licence recorded at download time.
	Provenance *schema.Provenance `json:"provenance,omitempty"`
}

// RepoFromSource extracts owner/name from a "hf:owner/name/file" source;
// empty for local files.
func RepoFromSource(source string) string {
	rest, ok := strings.CutPrefix(source, "hf:")
	if !ok {
		return ""
	}
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	return parts[0] + "/" + parts[1]
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
	ids := UniqueIDs(models)
	for i := range models {
		m := &models[i]
		id := ids[i]
		meta, ok := LoadMeta(dir, id)
		if !ok {
			meta = Meta{ID: id, Path: settings.RelativizeModelPath(m.Path), MMProj: settings.RelativizeModelPath(m.MMProj)}
			if SaveMeta(dir, meta) == nil {
				created++
			}
		}
		m.ID = id
		m.Alias = id
		if meta.Alias != "" {
			m.Alias = meta.Alias
		}
		m.Category = meta.Category
		m.Favorite = meta.Favorite
		m.LastUsed = meta.LastUsed
		m.MTP = meta.MTP
		m.Quant = meta.Quant
		if meta.Name != "" {
			m.Display = meta.Name
		}
		// Embedded MTP head + GGUF header details: trust the file over a
		// hand-written flag. A parse failure leaves the detail fields empty.
		if info, err := ReadGGUFInfo(m.Path); err == nil {
			if info.NextNPredictLayers > 0 {
				m.MTP = true
			}
			m.Arch = info.Arch
			m.Layers = info.BlockCount
			m.Heads = info.HeadCount
			m.KVHeads = info.HeadCountKV
			m.FullAttnLayers = info.FullAttentionLayers()
			m.TrainedContext = info.ContextLength
			m.FileSize = info.FileSize
			if m.Quant == "" {
				m.Quant = QuantFromName(m.Path)
			}
		}
		m.Source = meta.Source
		m.Provenance = meta.Provenance
		if m.Provenance == nil {
			if repo := RepoFromSource(meta.Source); repo != "" {
				m.Provenance = &schema.Provenance{Repo: repo}
			}
		}
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

// AliasFor returns the alias a model file is known by: the library override
// when one was set, otherwise the file-derived alias.
func AliasFor(dir, modelPath string) string {
	return AliasForID(dir, DerivedAlias(modelPath))
}

// AliasForID is AliasFor for an already resolved (possibly disambiguated) id.
func AliasForID(dir, id string) string {
	if meta, ok := LoadMeta(dir, id); ok && meta.Alias != "" {
		return meta.Alias
	}
	return id
}

// UpdateMeta loads (or stubs) the meta for modelPath, applies fn and saves it.
func UpdateMeta(dir, modelPath string, fn func(*Meta)) error {
	return UpdateMetaByID(dir, DerivedAlias(modelPath), modelPath, fn)
}

// UpdateMetaByID is UpdateMeta for a model with a stable (possibly hashed) id.
func UpdateMetaByID(dir, id, modelPath string, fn func(*Meta)) error {
	meta, ok := LoadMeta(dir, id)
	if !ok {
		meta = Meta{ID: id, Path: settings.RelativizeModelPath(modelPath)}
	}
	fn(&meta)
	return SaveMeta(dir, meta)
}

// DeleteMeta removes a model's library file (the .gguf itself is untouched).
func DeleteMeta(dir, id string) error {
	err := os.Remove(metaPath(dir, id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
