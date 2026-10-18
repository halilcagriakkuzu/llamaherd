package hf

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var shardRe = regexp.MustCompile(`-\d{5}-of-\d{5}\.gguf$`)

// IsModelFile reports whether a path is a loadable model (a .gguf that is not
// an mmproj sidecar and not a multi-part shard).
func IsModelFile(path string) bool {
	b := strings.ToLower(base(path))
	if !strings.HasSuffix(b, ".gguf") || strings.Contains(b, "mmproj") {
		return false
	}
	return !shardRe.MatchString(b)
}

func base(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// ResolveFile finds the requested file in a listing. want may be an exact
// path, a case-insensitive suffix ("Q4_K_M.gguf") or empty (the largest Q4_K_M
// model, else the largest model). mmproj sidecars and multi-part shards are
// never returned as the model.
func ResolveFile(tree []TreeFile, want string) (TreeFile, error) {
	want = strings.TrimSpace(want)
	if want != "" {
		if f, ok := matchFile(tree, want); ok {
			return f, nil
		}
		return TreeFile{}, fmt.Errorf("no file matching %q in this repository", want)
	}
	// Prefer a single Q4_K_M model; else the largest model.
	var models []TreeFile
	for _, f := range tree {
		if IsModelFile(f.Path) {
			models = append(models, f)
		}
	}
	if len(models) == 0 {
		return TreeFile{}, fmt.Errorf("repository contains no single-file .gguf model")
	}
	for _, f := range models {
		if strings.HasSuffix(strings.ToLower(f.Path), "q4_k_m.gguf") {
			return f, nil
		}
	}
	sort.Slice(models, func(i, j int) bool { return models[i].Size > models[j].Size })
	return models[0], nil
}

// matchFile finds want among the listing: exact path, then case-insensitive
// equal base name, then case-insensitive suffix, then substring.
func matchFile(tree []TreeFile, want string) (TreeFile, bool) {
	lw := strings.ToLower(want)
	for _, f := range tree {
		if f.Path == want {
			return f, true
		}
	}
	for _, f := range tree {
		if strings.EqualFold(base(f.Path), base(want)) {
			return f, true
		}
	}
	for _, f := range tree {
		if strings.HasSuffix(strings.ToLower(f.Path), strings.ToLower(base(want))) {
			return f, true
		}
	}
	for _, f := range tree {
		if strings.Contains(strings.ToLower(f.Path), lw) {
			return f, true
		}
	}
	return TreeFile{}, false
}

// IsMTPDraftFile reports whether a path looks like a separate MTP draft sidecar
// (e.g. HauhauCS "fastmtp*.gguf"). These are draft models for
// --spec-draft-model, not loadable target models.
func IsMTPDraftFile(path string) bool {
	b := strings.ToLower(base(path))
	if !strings.HasSuffix(b, ".gguf") {
		return false
	}
	for _, frag := range []string{"fastmtp", "mtp-draft", "mtp_draft", "draft-mtp", "draft_mtp"} {
		if strings.Contains(b, frag) {
			return true
		}
	}
	return false
}

// ResolveMMProj returns the vision projector for a repo: the requested file if
// given, else the smallest mmproj*.gguf (usually the F16 one is a safe default,
// but the smallest keeps VRAM low).
func ResolveMMProj(tree []TreeFile, want string) (TreeFile, bool) {
	if strings.TrimSpace(want) != "" {
		return matchFile(tree, want)
	}
	var mms []TreeFile
	for _, f := range tree {
		if strings.Contains(strings.ToLower(base(f.Path)), "mmproj") {
			mms = append(mms, f)
		}
	}
	if len(mms) == 0 {
		return TreeFile{}, false
	}
	sort.Slice(mms, func(i, j int) bool { return mms[i].Size < mms[j].Size })
	return mms[0], true
}

// ResolveMTPDraft returns a separate MTP draft sidecar from a listing, if any
// (want pins a specific file). Such sidecars need a draft-* spec type and, for
// FastMTP, a patched llama.cpp, so callers download and record them but leave
// them disabled in presets by default.
func ResolveMTPDraft(tree []TreeFile, want string) (TreeFile, bool) {
	if strings.TrimSpace(want) != "" {
		return matchFile(tree, want)
	}
	var drafts []TreeFile
	for _, f := range tree {
		if IsMTPDraftFile(f.Path) {
			drafts = append(drafts, f)
		}
	}
	if len(drafts) == 0 {
		return TreeFile{}, false
	}
	sort.Slice(drafts, func(i, j int) bool { return drafts[i].Size < drafts[j].Size })
	return drafts[0], true
}
