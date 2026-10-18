package hf

import (
	"strings"
)

// ParseRef splits a user-supplied model reference into repo and file. It
// accepts:
//
//	owner/name                                  (repo only; file is resolved)
//	owner/name/file.gguf                        (exact file)
//	owner/name/sub/dir/file.gguf                (nested file)
//	https://huggingface.co/owner/name/resolve/main/file.gguf
//	https://huggingface.co/owner/name/blob/main/file.gguf
//	hf:owner/name
//
// The returned file may be empty, meaning "pick the default model file".
func ParseRef(ref string) (repo, file string) {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "hf:")
	for _, p := range []string{
		"https://huggingface.co/", "http://huggingface.co/",
		"https://hf.co/", "http://hf.co/",
	} {
		ref = strings.TrimPrefix(ref, p)
	}
	// Drop a ?query or #fragment.
	if i := strings.IndexAny(ref, "?#"); i >= 0 {
		ref = ref[:i]
	}
	for _, marker := range []string{"/resolve/", "/blob/", "/raw/"} {
		if i := strings.Index(ref, marker); i >= 0 {
			rest := ref[i+len(marker):]
			// The segment after the marker is the revision (main, v1, …).
			if j := strings.Index(rest, "/"); j >= 0 {
				return ref[:i], rest[j+1:]
			}
			return ref[:i], ""
		}
	}
	parts := strings.Split(strings.Trim(ref, "/"), "/")
	if len(parts) >= 3 && strings.HasSuffix(strings.ToLower(parts[len(parts)-1]), ".gguf") {
		return strings.Join(parts[:2], "/"), strings.Join(parts[2:], "/")
	}
	if len(parts) >= 2 {
		return strings.Join(parts[:2], "/"), ""
	}
	return ref, ""
}
