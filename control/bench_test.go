package control

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunBenchmarkParsesTimings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/completion" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":"ok","timings":{"prompt_n":200,"prompt_ms":100,"prompt_per_second":2000,
			"predicted_n":64,"predicted_ms":1000,"predicted_per_second":64}}`))
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")

	s, err := RunBenchmark(context.Background(), addr, BenchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if s.PromptTokens != 200 || s.GenTokens != 64 || s.GenTokS != 64 || s.PromptTokS != 2000 {
		t.Fatalf("sample = %+v", s)
	}
}

func TestRunBenchmarkReportsMissingTimings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":"ok"}`))
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")
	if _, err := RunBenchmark(context.Background(), addr, BenchOptions{}); err == nil ||
		!strings.Contains(err.Error(), "timings") {
		t.Fatalf("err = %v, want a timings error", err)
	}
}
