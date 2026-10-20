package hf

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// fileSHA256 returns the lowercase hex sha256 of a file.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// short trims a hash for error messages.
func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// FetchFile resolves a repo file (and its size + sha256) in one call. want may
// be an exact path or a suffix; empty picks the default model file.
func FetchFile(ctx context.Context, repo, want, token string) (TreeFile, error) {
	tree, err := Tree(ctx, strings.TrimSpace(repo), token)
	if err != nil {
		return TreeFile{}, err
	}
	return ResolveFile(tree, want)
}

// FileSHA256 is the exported content hash used for duplicate detection.
func FileSHA256(path string) (string, error) { return fileSHA256(path) }
