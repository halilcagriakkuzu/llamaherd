package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// hfFile is one file served by the fake Hub.
type hfFile struct {
	Path    string
	Content []byte
	// Block, when set, holds the first transfer after half the body until the
	// channel is closed (used to pause a download mid-flight).
	Block chan struct{}
}

type downloadHub struct {
	mu     sync.Mutex
	reqs   []string
	ranges []string
	files  map[string][]byte
	block  map[string]chan struct{}
	tamper map[string]bool
}

func (h *downloadHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.reqs = append(h.reqs, r.URL.Path)
	if rg := r.Header.Get("Range"); rg != "" {
		h.ranges = append(h.ranges, r.URL.Path+" "+rg)
	}
	h.mu.Unlock()

	p := strings.TrimPrefix(r.URL.Path, "/")
	switch {
	case strings.Contains(p, "/tree/main"):
		repo := strings.TrimSuffix(p, "/tree/main")
		repo = strings.TrimPrefix(repo, "api/models/")
		h.mu.Lock()
		var entries []string
		for name, b := range h.files {
			if strings.HasPrefix(name, repo+"/") {
				file := strings.TrimPrefix(name, repo+"/")
				sum := sha256.Sum256(b)
				entries = append(entries, fmt.Sprintf(`{"path":%q,"type":"file","size":%d,"lfs":{"oid":%q,"size":%d}}`, file, len(b), hex.EncodeToString(sum[:]), len(b)))
			}
		}
		h.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "[%s]", strings.Join(entries, ","))
	case strings.HasPrefix(p, "api/models/"):
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"gated":false,"cardData":{"license":"mit"}}`)
	default:
		i := strings.Index(p, "/resolve/main/")
		if i < 0 {
			http.NotFound(w, r)
			return
		}
		key := p[:i] + "/" + p[i+len("/resolve/main/"):]
		h.mu.Lock()
		body, ok := h.files[key]
		gate := h.block[key]
		tamper := h.tamper[key]
		h.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		if tamper {
			body = append([]byte(nil), body...)
			body[0] ^= 0xff
		}
		if gate != nil && r.Header.Get("Range") == "" {
			half := len(body) / 2
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			w.Header().Set("Accept-Ranges", "bytes")
			_, _ = w.Write(body[:half])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			<-gate
			_, _ = w.Write(body[half:])
			return
		}
		http.ServeContent(w, r, filepath.Base(key), time.Time{}, bytes.NewReader(body))
	}
}

func (h *downloadHub) requests() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.reqs...)
}

func newDownloadApp(t *testing.T, hub *downloadHub) *App {
	t.Helper()
	srv := httptest.NewServer(hub)
	t.Cleanup(srv.Close)
	t.Setenv("HF_ENDPOINT", srv.URL)
	a, _ := libApp(t, "")
	return a
}

func waitDownload(t *testing.T, a *App, want func(DownloadState) bool) DownloadState {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st := a.DownloadSnapshot()
		if want(st) {
			return st
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("download did not reach the expected state: %+v", a.DownloadSnapshot())
	return DownloadState{}
}

func TestDownloadQueueRunsSequentiallyAndRemovesWaitingJob(t *testing.T) {
	hub := &downloadHub{
		files: map[string][]byte{
			"o/one/a.gguf": []byte(strings.Repeat("A", 4096)),
			"o/two/b.gguf": []byte(strings.Repeat("B", 4096)),
		},
		block: map[string]chan struct{}{"o/one/a.gguf": make(chan struct{})},
	}
	a := newDownloadApp(t, hub)
	if err := a.StartDownload("o/one", "a.gguf", "none", "none"); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, a, func(s DownloadState) bool { return s.Active && s.Repo == "o/one" })

	if err := a.StartDownload("o/two", "b.gguf", "none", "none"); err != nil {
		t.Fatal(err)
	}
	st := a.DownloadSnapshot()
	if len(st.Queue) != 1 || st.Queue[0].Repo != "o/two" {
		t.Fatalf("second download should be queued: %+v", st)
	}
	if err := a.RemoveQueuedDownload(st.Queue[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := a.RemoveQueuedDownload("dl-1"); err == nil {
		t.Error("removing the active download must fail")
	}
	close(hub.block["o/one/a.gguf"])
	waitDownload(t, a, func(s DownloadState) bool { return !s.Active && s.Status == "done" })
	for _, p := range hub.requests() {
		if strings.Contains(p, "/two/") {
			t.Fatalf("removed queued job must not download: %v", hub.requests())
		}
	}
}

func TestDownloadQueuePauseResumeUsesPartFile(t *testing.T) {
	content := []byte(strings.Repeat("llama", 4000))
	gate := make(chan struct{})
	hub := &downloadHub{
		files: map[string][]byte{"o/p/p.gguf": content},
		block: map[string]chan struct{}{"o/p/p.gguf": gate},
	}
	a := newDownloadApp(t, hub)
	if err := a.StartDownload("o/p", "p.gguf", "none", "none"); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, a, func(s DownloadState) bool { return s.Active && s.Received > 0 })
	part := filepath.Join(a.modelsDir, "o/p", "p.gguf.part")
	deadline := time.Now().Add(5 * time.Second)
	for {
		if st, err := os.Stat(part); err == nil && st.Size() > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no .part file written before pause")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := a.PauseDownload(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, a, func(s DownloadState) bool { return !s.Active && s.Status == "paused" })
	close(gate)
	if err := a.ResumeDownload(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, a, func(s DownloadState) bool { return !s.Active && s.Status == "done" })
	if _, err := os.Stat(filepath.Join(a.modelsDir, "o/p", "p.gguf")); err != nil {
		t.Fatalf("downloaded file missing: %v", err)
	}
	resumed := false
	hub.mu.Lock()
	for _, r := range hub.ranges {
		if strings.Contains(r, "p.gguf") && !strings.Contains(r, "bytes=0-") {
			resumed = true
		}
	}
	hub.mu.Unlock()
	if !resumed {
		t.Fatalf("resume did not use a Range request: %v", hub.ranges)
	}
}

func TestDownloadQueueTwoJobsRunInOrder(t *testing.T) {
	hub := &downloadHub{files: map[string][]byte{
		"o/a/a.gguf": []byte("aaa"),
		"o/b/b.gguf": []byte("bbb"),
	}}
	a := newDownloadApp(t, hub)
	if err := a.StartDownload("o/a", "a.gguf", "none", "none"); err != nil {
		t.Fatal(err)
	}
	if err := a.StartDownload("o/b", "b.gguf", "none", "none"); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, a, func(s DownloadState) bool {
		return !s.Active && s.Status == "done" && len(s.Queue) == 0
	})
	var got []string
	for _, p := range hub.requests() {
		if strings.Contains(p, "/resolve/main/") {
			got = append(got, p)
		}
	}
	if len(got) != 2 || !strings.Contains(got[0], "/a/") || !strings.Contains(got[1], "/b/") {
		t.Fatalf("download order = %v", got)
	}
}

func TestMultiPartDownloadInstallsFirstShard(t *testing.T) {
	hub := &downloadHub{files: map[string][]byte{
		"o/set/Model-00001-of-00002.gguf": []byte(strings.Repeat("1", 100)),
		"o/set/Model-00002-of-00002.gguf": []byte(strings.Repeat("2", 100)),
	}}
	a := newDownloadApp(t, hub)
	if err := a.StartDownload("o/set", "Model-00001-of-00002.gguf", "none", "none"); err != nil {
		t.Fatal(err)
	}
	st := waitDownload(t, a, func(s DownloadState) bool {
		return !s.Active && s.Status == "done" && len(s.Queue) == 0
	})
	if len(st.Files) != 2 {
		t.Fatalf("downloaded files = %v", st.Files)
	}
	for name := range hub.files {
		dest := filepath.Join(a.modelsDir, "o/set", filepath.Base(name))
		if _, err := os.Stat(dest); err != nil {
			t.Fatalf("shard missing: %v", err)
		}
	}
	first := filepath.Join(a.modelsDir, "o/set", "Model-00001-of-00002.gguf")
	i := a.indexOfModelPath(first)
	if i < 0 {
		t.Fatalf("first shard not scanned as a model: %+v", a.models)
	}
	list := a.presets[a.models[i].Alias]
	if len(list) == 0 || !strings.HasSuffix(list[0].Basic.Model, "Model-00001-of-00002.gguf") {
		t.Fatalf("preset must load the first shard: %+v", list)
	}
}

func TestMultiPartDownloadRejectsCorruptShard(t *testing.T) {
	hub := &downloadHub{
		files: map[string][]byte{
			"o/bad/Model-00001-of-00002.gguf": []byte(strings.Repeat("1", 100)),
			"o/bad/Model-00002-of-00002.gguf": []byte(strings.Repeat("2", 100)),
		},
		tamper: map[string]bool{"o/bad/Model-00002-of-00002.gguf": true},
	}
	a := newDownloadApp(t, hub)
	if err := a.StartDownload("o/bad", "Model-00001-of-00002.gguf", "none", "none"); err != nil {
		t.Fatal(err)
	}
	st := waitDownload(t, a, func(s DownloadState) bool {
		return !s.Active && s.Status == "error"
	})
	if !strings.Contains(st.Error, "checksum mismatch") {
		t.Fatalf("error = %q", st.Error)
	}
	if _, err := os.Stat(filepath.Join(a.modelsDir, "o/bad", "Model-00002-of-00002.gguf")); !os.IsNotExist(err) {
		t.Fatal("corrupt shard must not be installed")
	}
	for _, m := range a.models {
		if strings.Contains(m.Path, "o/bad") || strings.Contains(m.Path, "o"+string(filepath.Separator)+"bad") {
			t.Fatalf("corrupt set must not create a model: %+v", m)
		}
	}
}

func TestMultiPartDownloadRejectsMissingShard(t *testing.T) {
	hub := &downloadHub{files: map[string][]byte{
		"o/gap/Model-00001-of-00002.gguf": []byte(strings.Repeat("1", 100)),
	}}
	a := newDownloadApp(t, hub)
	if err := a.StartDownload("o/gap", "Model-00001-of-00002.gguf", "none", "none"); err != nil {
		t.Fatal(err)
	}
	st := waitDownload(t, a, func(s DownloadState) bool { return !s.Active && s.Status == "error" })
	if !strings.Contains(st.Error, "missing") {
		t.Fatalf("error = %q", st.Error)
	}
}
