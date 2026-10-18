package hf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"llamaherd/settings"
)

// Endpoint returns the Hub base URL. HF_ENDPOINT overrides it (mirrors and
// tests).
func Endpoint() string {
	if v := strings.TrimSpace(os.Getenv("HF_ENDPOINT")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://huggingface.co"
}

// Token returns the Hugging Face access token (env, then the llamaherd token
// file, then the standard HF cache). Empty means anonymous.
func Token() string { return settings.HFToken() }

// FileURL is the resolve URL for repo/file at the main revision.
func FileURL(repo, file string) string {
	return Endpoint() + "/" + repo + "/resolve/main/" + escapePath(file)
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

// TreeFile is one file entry from the Hub file listing. OID is the sha256 of
// an LFS file (empty for small non-LFS files).
type TreeFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	OID  string `json:"-"`
}

type treeJSON struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size"`
	LFS  *struct {
		OID  string `json:"oid"`
		Size int64  `json:"size"`
	} `json:"lfs"`
}

// Tree lists the files of a repository at the main revision.
func Tree(ctx context.Context, repo, token string) ([]TreeFile, error) {
	u := Endpoint() + "/api/models/" + repo + "/tree/main?recursive=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("repository %s requires authentication (set a Hugging Face token)", repo)
	case http.StatusNotFound:
		return nil, fmt.Errorf("repository not found: %s", repo)
	default:
		return nil, fmt.Errorf("hub returned %s for %s", resp.Status, repo)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var raw []treeJSON
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("unexpected hub response for %s: %w", repo, err)
	}
	out := make([]TreeFile, 0, len(raw))
	for _, f := range raw {
		if f.Type != "file" {
			continue
		}
		tf := TreeFile{Path: f.Path, Size: f.Size}
		if f.LFS != nil {
			tf.OID = f.LFS.OID
			if f.LFS.Size > 0 {
				tf.Size = f.LFS.Size
			}
		}
		out = append(out, tf)
	}
	return out, nil
}
