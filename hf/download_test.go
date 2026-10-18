package hf

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var timeZero = time.Time{}

func serve(t *testing.T, content []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "file.gguf", timeZero, bytes.NewReader(content))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("HF_ENDPOINT", srv.URL)
	return srv
}

func TestDownloadResume(t *testing.T) {
	content := bytes.Repeat([]byte("llama-herd!"), 100000) // ~1.1 MB
	sum := sha256.Sum256(content)
	hexSum := hex.EncodeToString(sum[:])
	serve(t, content)

	dir := t.TempDir()
	dest := filepath.Join(dir, "file.gguf")
	if err := os.WriteFile(PartPath(dest), content[:len(content)/2], 0o644); err != nil {
		t.Fatal(err)
	}

	var phases []string
	got, err := Download(context.Background(), Options{
		Repo: "org/repo", File: "file.gguf", DestDir: dir,
		ExpectedSize: int64(len(content)), ExpectedSHA256: hexSum,
		OnProgress: func(p Progress) { phases = append(phases, p.Phase) },
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(got)
	if !bytes.Equal(b, content) {
		t.Fatalf("content mismatch (%d vs %d bytes)", len(b), len(content))
	}
	if _, err := os.Stat(PartPath(dest)); !os.IsNotExist(err) {
		t.Fatal(".part should be gone after a successful download")
	}
	if len(phases) == 0 {
		t.Fatal("no progress callbacks fired")
	}
}

func TestDownloadChecksumMismatch(t *testing.T) {
	serve(t, []byte("hello world"))
	dir := t.TempDir()
	_, err := Download(context.Background(), Options{
		Repo: "org/repo", File: "file.gguf", DestDir: dir,
		ExpectedSHA256: "deadbeef00000000000000000000000000000000000000000000000000000000",
	})
	if err == nil {
		t.Fatal("expected a checksum error")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "file.gguf.part")); !os.IsNotExist(statErr) {
		t.Fatal("corrupt .part should be removed")
	}
}
