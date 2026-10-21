package control

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseMetrics(t *testing.T) {
	text := `
# HELP llamacpp:prompt_seconds_total prompt eval seconds
llamacpp:prompt_seconds_total 1.25
llamacpp:requests_processing 2
llamacpp:slots_busy 1
llamacpp:requests_deferred 3
llamacpp:kv_cache_tokens 4096
llamacpp:kv_cache_usage_ratio 0.25
`
	s := parseMetrics(text)
	if s.PromptSecondsTotal != 1.25 || s.SlotsActive != 2 || s.SlotsBusy != 1 || s.QueueDepth != 3 || s.ContextTokens != 4096 || s.KVCacheRatio != 0.25 {
		t.Fatalf("metrics = %+v", s)
	}
}

func TestScrapeMetricsOlderBuild(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")
	if _, ok := ScrapeMetrics(addr); ok {
		t.Fatal("a 404 /metrics must report unavailable")
	}
}

func TestRunBenchmarkCounters(t *testing.T) {
	before := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metrics":
			if before {
				before = false
				_, _ = w.Write([]byte("llamacpp:prompt_seconds_total 1.0\n"))
				return
			}
			_, _ = w.Write([]byte("llamacpp:prompt_seconds_total 1.4\nllamacpp:requests_processing 1\nllamacpp:slots_busy 1\nllamacpp:requests_deferred 0\nllamacpp:kv_cache_tokens 214\n"))
		case "/completion":
			_, _ = w.Write([]byte(`{"timings":{"prompt_n":200,"prompt_ms":100,"prompt_per_second":2000,"predicted_n":64,"predicted_ms":1000,"predicted_per_second":64}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")
	s, err := RunBenchmark(t.Context(), addr, BenchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if s.PromptEvalMS < 399 || s.PromptEvalMS > 401 {
		t.Errorf("prompt eval delta = %f, want ~400ms", s.PromptEvalMS)
	}
	if s.SlotsActive != 1 || s.SlotsBusy != 1 || s.QueueDepth != 0 || s.ContextTokens != 214 {
		t.Errorf("counters = %+v", s)
	}
}
