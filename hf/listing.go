package hf

import (
	"context"
	"regexp"
	"sort"
	"strings"
)

// RepoFile is one downloadable .gguf of a repository.
type RepoFile struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Quant string `json:"quant,omitempty"` // e.g. "UD-Q4_K_XL", "BF16"
	Size  int64  `json:"size"`
	Parts int    `json:"parts,omitempty"` // >1: a multi-part shard set
	// Shards lists every shard path of a multi-part set, in order (first shard
	// first). Empty for a single-file model.
	Shards []string `json:"shards,omitempty"`
}

// RepoListing groups a repository's .gguf files by role so a UI can offer them
// as a choice: loadable models (smallest first), vision projectors and MTP
// draft sidecars.
type RepoListing struct {
	Repo string `json:"repo"`
	// Gated/License come from the model API (empty when the lookup failed); a
	// gated repository needs its licence accepted on Hugging Face first.
	Gated     string     `json:"gated,omitempty"`
	License   string     `json:"license,omitempty"`
	Models    []RepoFile `json:"models"`
	MMProjs   []RepoFile `json:"mmprojs"`
	MTPDrafts []RepoFile `json:"mtpDrafts"`
}

var (
	quantRe   = regexp.MustCompile(`(?i)(UD-)?(PTQ|PQ|TQ|IQ|Q)\d(?:_[A-Z0-9]+)*|MXFP4(?:_MOE)?|BF16|F16|F32`)
	shardName = regexp.MustCompile(`(?i)^(.*?)-(\d{5})-of-(\d{5})\.gguf$`)
)

// QuantOf extracts the quantization label from a GGUF file name ("" if none).
func QuantOf(name string) string {
	name = strings.TrimSuffix(name, ".gguf")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	m := quantRe.FindAllString(name, -1)
	if len(m) == 0 {
		return ""
	}
	return strings.ToUpper(m[len(m)-1])
}

// ListRepo fetches a repository's file list and classifies its .gguf files.
// Multi-part shard sets are returned as a single entry with Parts > 1 and the
// summed size (the downloader handles single files only).
func ListRepo(ctx context.Context, ref, token string) (RepoListing, error) {
	repo, _, err := ParseRef(ref)
	if err != nil {
		return RepoListing{}, err
	}
	tree, err := Tree(ctx, repo, token)
	if err != nil {
		return RepoListing{}, err
	}
	out := classify(repo, tree)
	if info, err := FetchModelInfo(ctx, repo, token); err == nil {
		out.Gated, out.License = info.Gated, info.License
	}
	return out, nil
}

func classify(repo string, tree []TreeFile) RepoListing {
	out := RepoListing{Repo: repo, Models: []RepoFile{}, MMProjs: []RepoFile{}, MTPDrafts: []RepoFile{}}
	shards := map[string][]TreeFile{}
	for _, f := range tree {
		low := strings.ToLower(f.Path)
		if !strings.HasSuffix(low, ".gguf") {
			continue
		}
		rf := RepoFile{Path: f.Path, Name: base(f.Path), Quant: QuantOf(f.Path), Size: f.Size}
		switch {
		case strings.Contains(strings.ToLower(rf.Name), "mmproj"):
			out.MMProjs = append(out.MMProjs, rf)
		case IsMTPDraftFile(f.Path):
			out.MTPDrafts = append(out.MTPDrafts, rf)
		case shardName.MatchString(rf.Name):
			si, _ := ParseShard(f.Path)
			k := si.Dir + strings.ToLower(si.Base)
			shards[k] = append(shards[k], f)
		case strings.Contains(strings.ToLower(rf.Name), "imatrix"):
			// calibration data, not a model
		default:
			out.Models = append(out.Models, rf)
		}
	}
	for _, set := range shards {
		files := sortShards(set)
		first := files[0]
		si, _ := ParseShard(first.Path)
		rf := RepoFile{
			Path:   first.Path,
			Name:   base(si.Base) + ".gguf",
			Quant:  QuantOf(first.Path),
			Parts:  si.Total,
			Shards: make([]string, len(files)),
		}
		for i, f := range files {
			rf.Size += f.Size
			rf.Shards[i] = f.Path
		}
		out.Models = append(out.Models, rf)
	}
	by := func(l []RepoFile) {
		sort.Slice(l, func(i, j int) bool {
			if l[i].Size != l[j].Size {
				return l[i].Size < l[j].Size
			}
			return l[i].Path < l[j].Path
		})
	}
	by(out.Models)
	by(out.MMProjs)
	by(out.MTPDrafts)
	return out
}
