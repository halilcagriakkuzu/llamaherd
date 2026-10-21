package hf

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var shardRe = regexp.MustCompile(`-\d{5}-of-\d{5}\.gguf$`)

// ShardInfo describes one part of a multi-part GGUF set. Dir includes the
// trailing slash; Base is the file prefix shared by every shard.
type ShardInfo struct {
	Dir   string
	Base  string
	Index int
	Total int
}

// ParseShard splits a shard file name ("model-00001-of-00002.gguf").
func ParseShard(p string) (ShardInfo, bool) {
	b := base(p)
	m := shardName.FindStringSubmatch(b)
	if m == nil {
		return ShardInfo{}, false
	}
	idx, err1 := strconv.Atoi(m[2])
	total, err2 := strconv.Atoi(m[3])
	if err1 != nil || err2 != nil || idx < 1 || total < 1 {
		return ShardInfo{}, false
	}
	dir := ""
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		dir = p[:i+1]
	}
	return ShardInfo{Dir: dir, Base: m[1], Index: idx, Total: total}, true
}

// sortShards orders a shard set by part index.
func sortShards(files []TreeFile) []TreeFile {
	out := append([]TreeFile(nil), files...)
	sort.Slice(out, func(i, j int) bool {
		a, _ := ParseShard(out[i].Path)
		b, _ := ParseShard(out[j].Path)
		return a.Index < b.Index
	})
	return out
}

// ResolveModelSet resolves a model selection to one or more files. want may be
// a single-file model (returns one TreeFile) or any shard / shard-set name of a
// multi-part set (returns every shard, first part first). The shard count is
// verified so a missing part fails before anything is installed.
func ResolveModelSet(tree []TreeFile, want string) ([]TreeFile, error) {
	want = strings.TrimSpace(want)
	if want == "" {
		f, err := ResolveFile(tree, "")
		if err != nil {
			return nil, err
		}
		return []TreeFile{f}, nil
	}
	// A candidate shard: exact path, case-insensitive base or suffix.
	for _, f := range tree {
		si, ok := ParseShard(f.Path)
		if !ok {
			continue
		}
		if f.Path == want || strings.EqualFold(base(f.Path), base(want)) ||
			strings.HasSuffix(strings.ToLower(f.Path), strings.ToLower(base(want))) {
			return shardSet(tree, si)
		}
	}
	// The listing names a set by its prefix ("model.gguf"); accept that too.
	for _, f := range tree {
		si, ok := ParseShard(f.Path)
		if !ok {
			continue
		}
		if strings.EqualFold(base(want), base(si.Base)+".gguf") ||
			strings.EqualFold(want, si.Dir+base(si.Base)+".gguf") {
			return shardSet(tree, si)
		}
	}
	f, err := ResolveFile(tree, want)
	if err != nil {
		return nil, err
	}
	return []TreeFile{f}, nil
}

// shardSet collects and validates every shard of the set described by want.
func shardSet(tree []TreeFile, want ShardInfo) ([]TreeFile, error) {
	var files []TreeFile
	for _, f := range tree {
		si, ok := ParseShard(f.Path)
		if !ok {
			continue
		}
		if si.Dir == want.Dir && strings.EqualFold(si.Base, want.Base) {
			files = append(files, f)
		}
	}
	files = sortShards(files)
	if len(files) != want.Total {
		return nil, fmt.Errorf("multi-part model %s has %d of %d shards (a part is missing)", want.Base, len(files), want.Total)
	}
	for i, f := range files {
		si, _ := ParseShard(f.Path)
		if si.Index != i+1 {
			return nil, fmt.Errorf("multi-part model %s has a gap in its shards (expected part %d, found %d)", want.Base, i+1, si.Index)
		}
	}
	return files, nil
}

// IsModelFile reports whether a path is a loadable model (a .gguf that is not
// an mmproj sidecar and not a multi-part shard).
func IsModelFile(path string) bool {
	b := strings.ToLower(base(path))
	if !strings.HasSuffix(b, ".gguf") || strings.Contains(b, "mmproj") || IsMTPDraftFile(path) {
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
	// unsloth ships its MTP heads as "MTP/mtp-<model>-<quant>.gguf"
	if strings.HasPrefix(strings.ToLower(path), "mtp/") || strings.HasPrefix(b, "mtp-") {
		return true
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
	if IsNone(want) {
		return TreeFile{}, false
	}
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
	if IsNone(want) {
		return TreeFile{}, false
	}
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

// None is the sentinel for "do not download this optional sidecar" (vision
// projector or MTP draft); the empty string means "use it when the repo has one".
const None = "none"

// IsNone reports whether an optional-sidecar choice means "skip it".
func IsNone(want string) bool { return strings.EqualFold(strings.TrimSpace(want), None) }
