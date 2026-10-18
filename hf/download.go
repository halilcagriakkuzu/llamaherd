package hf

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Progress is a download progress snapshot.
type Progress struct {
	Received int64   `json:"received"`
	Total    int64   `json:"total"`
	Percent  float64 `json:"percent"`
	SpeedBps float64 `json:"speedBps"`
	Phase    string  `json:"phase"` // downloading | verifying
}

// Options describe one file download.
type Options struct {
	Repo           string
	File           string
	DestDir        string
	Token          string
	ExpectedSize   int64  // 0 = unknown
	ExpectedSHA256 string // "" = skip checksum
	Client         *http.Client
	OnProgress     func(Progress)
	Retries        int // extra attempts on transient errors
}

// PartPath is the in-progress file path for a destination.
func PartPath(dest string) string { return dest + ".part" }

// errChecksum marks a sha256 mismatch (not retried; the .part is removed).
var errChecksum = errors.New("checksum mismatch")

// Download fetches one file into DestDir, resuming a previous .part if present
// and verifying size + sha256 before atomically renaming it into place. It
// returns the final path. A cancelled or failed download keeps the .part file
// for a later resume; a checksum mismatch removes the corrupt .part.
func Download(ctx context.Context, opts Options) (string, error) {
	if opts.Repo == "" || opts.File == "" {
		return "", fmt.Errorf("repo and file are required")
	}
	if opts.DestDir == "" {
		return "", fmt.Errorf("destination directory is required")
	}
	if err := os.MkdirAll(opts.DestDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(opts.DestDir, filepath.Base(opts.File))
	part := PartPath(dest)
	client := opts.Client
	if client == nil {
		client = &http.Client{} // no overall timeout: large bodies stream
	}

	var lastErr error
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		lastErr = downloadOnce(ctx, client, opts, dest, part)
		if lastErr == nil {
			return dest, nil
		}
		if ctx.Err() != nil || errors.Is(lastErr, errChecksum) {
			return "", lastErr
		}
		if attempt < opts.Retries {
			time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
		}
	}
	return "", lastErr
}
