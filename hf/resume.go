package hf

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// downloadOnce performs a single HTTP transfer, resuming from an existing
// .part file when the server honours a Range request.
func downloadOnce(ctx context.Context, client *http.Client, opts Options, dest, part string) error {
	var offset int64
	if st, err := os.Stat(part); err == nil {
		offset = st.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, FileURL(opts.Repo, opts.File), nil)
	if err != nil {
		return err
	}
	if opts.Token != "" {
		req.Header.Set("Authorization", "Bearer "+opts.Token)
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("access denied for %s (set a Hugging Face token for private/gated repos)", opts.File)
	case http.StatusNotFound:
		return fmt.Errorf("file not found: %s/%s", opts.Repo, opts.File)
	default:
		return fmt.Errorf("download of %s failed: %s", opts.File, resp.Status)
	}

	partial := resp.StatusCode == http.StatusPartialContent
	if !partial {
		offset = 0 // server ignored Range: restart from scratch
	}
	total := opts.ExpectedSize
	if resp.ContentLength > 0 {
		total = offset + resp.ContentLength
	}

	flag := os.O_CREATE | os.O_WRONLY
	if partial {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	f, err := os.OpenFile(part, flag, 0o644)
	if err != nil {
		return err
	}

	received := offset
	base := received
	start := time.Now()
	buf := make([]byte, 256*1024)
	for {
		if err := ctx.Err(); err != nil {
			f.Close()
			return err
		}
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return werr
			}
			received += int64(n)
			emit(opts.OnProgress, received, total, base, start, "downloading")
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return rerr
		}
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	if total > 0 && received != total {
		return fmt.Errorf("incomplete download of %s: got %d of %d bytes", opts.File, received, total)
	}
	if opts.ExpectedSHA256 != "" {
		emit(opts.OnProgress, received, total, base, start, "verifying")
		sum, err := fileSHA256(part)
		if err != nil {
			return err
		}
		if !strings.EqualFold(sum, opts.ExpectedSHA256) {
			os.Remove(part)
			return fmt.Errorf("%w for %s (expected %s, got %s)", errChecksum, opts.File, short(opts.ExpectedSHA256), short(sum))
		}
	}
	return os.Rename(part, dest)
}

func emit(cb func(Progress), received, total, base int64, start time.Time, phase string) {
	if cb == nil {
		return
	}
	p := Progress{Received: received, Total: total, Phase: phase}
	if total > 0 {
		p.Percent = float64(received) / float64(total) * 100
	}
	if d := time.Since(start).Seconds(); d > 0 {
		p.SpeedBps = float64(received-base) / d
	}
	cb(p)
}
