package hf

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"unicode"
)

// repoPartRe matches one owner/name segment: Hugging Face allows letters,
// digits, dots, hyphens and underscores. The first character must be
// alphanumeric (no hidden ".git"-style names, no empty segments).
var repoPartRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// maxRepoPart is the Hugging Face limit for a repo/owner segment.
const maxRepoPart = 96

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
// Malformed references (path traversal, backslashes, control characters,
// invalid owner/name) are rejected instead of being passed to the downloader.
func ParseRef(ref string) (repo, file string, err error) {
	raw := strings.TrimSpace(ref)
	if raw == "" {
		return "", "", fmt.Errorf("empty model reference")
	}
	if strings.ContainsRune(raw, '\\') {
		return "", "", fmt.Errorf("invalid model reference: backslashes are not allowed")
	}
	if hasControlRune(raw) {
		return "", "", fmt.Errorf("invalid model reference: control characters are not allowed")
	}
	// Drop a URL query or fragment before decoding.
	if i := strings.IndexAny(raw, "?#"); i >= 0 {
		raw = raw[:i]
	}
	for _, p := range []string{
		"https://huggingface.co/", "http://huggingface.co/",
		"https://hf.co/", "http://hf.co/",
	} {
		raw = strings.TrimPrefix(raw, p)
	}
	raw = strings.TrimPrefix(raw, "hf:")
	if strings.HasPrefix(raw, "/") {
		return "", "", fmt.Errorf("invalid model reference %q: use owner/name", raw)
	}

	repoRaw, fileRaw := raw, ""
	for _, marker := range []string{"/resolve/", "/blob/", "/raw/"} {
		if i := strings.Index(raw, marker); i >= 0 {
			rest := raw[i+len(marker):]
			// The segment after the marker is the revision (main, v1, …).
			if j := strings.Index(rest, "/"); j >= 0 {
				repoRaw, fileRaw = raw[:i], rest[j+1:]
			} else {
				repoRaw = raw[:i]
			}
			break
		}
	}
	if fileRaw == "" {
		parts := strings.Split(strings.Trim(raw, "/"), "/")
		switch {
		case len(parts) == 2:
			repoRaw = strings.Join(parts, "/")
		case len(parts) > 2 && strings.HasSuffix(strings.ToLower(parts[len(parts)-1]), ".gguf"):
			repoRaw, fileRaw = strings.Join(parts[:2], "/"), strings.Join(parts[2:], "/")
		case len(parts) > 2:
			return "", "", fmt.Errorf("invalid model reference %q: a file path must end in .gguf", raw)
		}
	}

	repoRaw, err = decodeRefPart(repoRaw)
	if err != nil {
		return "", "", err
	}
	repo, err = validateRepo(repoRaw)
	if err != nil {
		return "", "", err
	}
	if fileRaw != "" {
		fileRaw, err = decodeRefPart(fileRaw)
		if err != nil {
			return "", "", err
		}
		file, err = validateRefFile(fileRaw)
		if err != nil {
			return "", "", err
		}
	}
	return repo, file, nil
}

// decodeRefPart percent-decodes one piece of the reference. Decoding before
// validation is what makes "%2e%2e" visible as "..".
func decodeRefPart(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("invalid model reference: empty repository")
	}
	dec, err := url.PathUnescape(s)
	if err != nil {
		return "", fmt.Errorf("invalid model reference: %w", err)
	}
	if hasControlRune(dec) || strings.ContainsRune(dec, '\\') {
		return "", fmt.Errorf("invalid model reference: forbidden characters")
	}
	return dec, nil
}

// validateRepo checks that s is exactly owner/name with safe segments.
func validateRepo(s string) (string, error) {
	s = strings.Trim(s, "/")
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid model reference %q: expected owner/name", s)
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." || len(p) > maxRepoPart || !repoPartRe.MatchString(p) {
			return "", fmt.Errorf("invalid model reference %q: owner and name may only contain letters, digits, '.', '-' and '_'", s)
		}
	}
	return strings.Join(parts, "/"), nil
}

// validateRefFile rejects absolute paths, traversal, empty segments and
// anything that is not a clean relative slash-separated path.
func validateRefFile(s string) (string, error) {
	if strings.HasPrefix(s, "/") {
		return "", fmt.Errorf("invalid file path %q: must be relative", s)
	}
	if path.Clean(s) != s || s == "." {
		return "", fmt.Errorf("invalid file path %q: must be a clean relative path", s)
	}
	for _, seg := range strings.Split(s, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", fmt.Errorf("invalid file path %q: path traversal is not allowed", s)
		}
		for _, r := range seg {
			if r == '\x00' || unicode.IsControl(r) {
				return "", fmt.Errorf("invalid file path %q: control characters are not allowed", s)
			}
		}
	}
	return s, nil
}

func hasControlRune(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// ValidateFileRef percent-decodes and validates a file path supplied by an API
// or CLI caller (the download endpoint accepts a file separately from the
// repo). An empty file is rejected; callers should skip the call when none was
// supplied.
func ValidateFileRef(s string) (string, error) {
	dec, err := decodeRefPart(s)
	if err != nil {
		return "", err
	}
	return validateRefFile(dec)
}
