package control

// bench.go — a small, reproducible speed probe against a running llama-server.
// It uses /completion (not the chat API) because that endpoint reports the
// server's own timings object, which is exactly what a benchmark table needs.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// BenchOptions controls the probe. Zero values fall back to the defaults.
type BenchOptions struct {
	Prompt    string
	MaxTokens int
	Timeout   time.Duration
}

// BenchSample is the measured result of one probe.
type BenchSample struct {
	PromptTokens int
	PromptMS     float64
	PromptTokS   float64
	GenTokens    int
	GenMS        float64
	GenTokS      float64
	DurationMS   int64
}

// DefaultBenchPrompt is deliberately fixed: every run must send the same input
// so the stored numbers are comparable across models and configs.
const DefaultBenchPrompt = `Explain, in a clear and compact way, how a large language model
generates text. Cover tokenization, the transformer forward pass, attention,
sampling and the role of the KV cache. Use plain language and a short example.
End with three practical tips for running a model on a desktop computer with a
consumer GPU.`

// RunBenchmark sends one completion request to the server at addr and returns
// its timings. addr is "host:port" (no scheme).
func RunBenchmark(ctx context.Context, addr string, opts BenchOptions) (BenchSample, error) {
	prompt := opts.Prompt
	if strings.TrimSpace(prompt) == "" {
		prompt = DefaultBenchPrompt
	}
	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 128
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	body, err := json.Marshal(map[string]any{
		"prompt":       prompt,
		"n_predict":    maxTokens,
		"temperature":  0,
		"top_k":        1,
		"top_p":        1,
		"cache_prompt": false,
		"stream":       false,
	})
	if err != nil {
		return BenchSample{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://"+addr+"/completion", bytes.NewReader(body))
	if err != nil {
		return BenchSample{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	started := time.Now()
	cl := &http.Client{Timeout: timeout}
	resp, err := cl.Do(req)
	if err != nil {
		return BenchSample{}, fmt.Errorf("benchmark request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return BenchSample{}, fmt.Errorf("benchmark: server returned %s", resp.Status)
	}
	var out struct {
		Timings *struct {
			PromptN         int     `json:"prompt_n"`
			PromptMS        float64 `json:"prompt_ms"`
			PromptPerSec    float64 `json:"prompt_per_second"`
			PredictedN      int     `json:"predicted_n"`
			PredictedMS     float64 `json:"predicted_ms"`
			PredictedPerSec float64 `json:"predicted_per_second"`
		} `json:"timings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return BenchSample{}, fmt.Errorf("benchmark: decode response: %w", err)
	}
	if out.Timings == nil || out.Timings.PredictedN <= 0 {
		return BenchSample{}, fmt.Errorf("benchmark: server did not report timings (a recent llama.cpp build is required)")
	}
	s := BenchSample{
		PromptTokens: out.Timings.PromptN,
		PromptMS:     out.Timings.PromptMS,
		PromptTokS:   out.Timings.PromptPerSec,
		GenTokens:    out.Timings.PredictedN,
		GenMS:        out.Timings.PredictedMS,
		GenTokS:      out.Timings.PredictedPerSec,
		DurationMS:   time.Since(started).Milliseconds(),
	}
	if s.PromptTokS <= 0 && s.PromptMS > 0 {
		s.PromptTokS = float64(s.PromptTokens) / (s.PromptMS / 1000)
	}
	if s.GenTokS <= 0 && s.GenMS > 0 {
		s.GenTokS = float64(s.GenTokens) / (s.GenMS / 1000)
	}
	return s, nil
}
