package control

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// VramMB reports GPU memory. In WSL the per-process query answers
// "[N/A]" (a known WSL nvidia-smi limitation), so we fall back to the
// total GPU memory used — good enough on a single-user workstation. A provider
// without a per-process number (AMD/Intel/Apple) uses the same fallback.
func VramMB(pid int) int {
	if pid <= 0 {
		return 0
	}
	p := ActiveGPU()
	if total := p.ProcessVRAMMB(pid); total > 0 {
		return total
	}
	return p.UsedVRAMMB()
}

// RamMB reports the resident set size of the given pid in MiB (0 when the
// platform cannot report it). Implementation is platform-specific
// (proc_linux.go / proc_darwin.go / proc_windows.go).
func RamMB(pid int) int {
	if pid <= 0 {
		return 0
	}
	return procRSS(pid)
}

// ServerProps is what a running llama-server reports about the model it has
// actually loaded (queried from /props). This is authoritative: it describes
// the running process, not whatever config the UI currently has selected.
type ServerProps struct {
	ModelPath  string
	ModelAlias string
	Ftype      string
	NCtx       int
	Slots      int
}

// FetchServerProps reads /props and returns the loaded model's identity and
// runtime parameters. Zero values mean "unknown" (server not up / old build).
func FetchServerProps(addr string) ServerProps {
	cl := &http.Client{Timeout: 2 * time.Second}
	resp, err := cl.Get("http://" + addr + "/props")
	if err != nil {
		return ServerProps{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ServerProps{}
	}
	var props struct {
		DefaultGenerationSettings struct {
			NCtx int `json:"n_ctx"`
		} `json:"default_generation_settings"`
		NCtx       int    `json:"n_ctx"`
		ModelPath  string `json:"model_path"`
		ModelAlias string `json:"model_alias"`
		ModelFtype string `json:"model_ftype"`
		TotalSlots int    `json:"total_slots"`
	}
	if json.NewDecoder(resp.Body).Decode(&props) != nil {
		return ServerProps{}
	}
	p := ServerProps{
		ModelPath:  props.ModelPath,
		ModelAlias: props.ModelAlias,
		Ftype:      props.ModelFtype,
		Slots:      props.TotalSlots,
	}
	if props.DefaultGenerationSettings.NCtx > 0 {
		p.NCtx = props.DefaultGenerationSettings.NCtx
	} else {
		p.NCtx = props.NCtx
	}
	return p
}

// NCtx fetches the server's actual context size from /props, or 0 on any
// failure (the caller falls back to the configured value).
func NCtx(addr string) int {
	return FetchServerProps(addr).NCtx
}

// HumanMB formats a memory amount the way the status panel shows it
// ("16.8 GB"); <=0 returns an em-dash.
func HumanMB(mb int) string {
	if mb <= 0 {
		return "—"
	}
	if mb < 1024 {
		return fmt.Sprintf("%d MB", mb)
	}
	return fmt.Sprintf("%.1f GB", float64(mb)/1024)
}

// GPUUtilTemp returns the highest utilization (%) and temperature (°C) across
// visible GPUs. Zero values mean "unavailable" (no provider, driver or WSL
// restrictions).
func GPUUtilTemp() (util, temp int) {
	return ActiveGPU().UtilTemp()
}

// MetricsSnapshot is the subset of llama-server's Prometheus /metrics endpoint
// useful for a benchmark. Zero values mean "not reported" (older build or
// --metrics off).
type MetricsSnapshot struct {
	PromptSecondsTotal float64
	SlotsActive        int
	SlotsBusy          int
	QueueDepth         int
	ContextTokens      int
	KVCacheRatio       float64
}

// ScrapeMetrics reads /metrics from a running llama-server. ok=false means the
// endpoint is unavailable (an older build, --metrics off, or a network error);
// callers must treat the counters as empty rather than failing.
func ScrapeMetrics(addr string) (MetricsSnapshot, bool) {
	cl := &http.Client{Timeout: 2 * time.Second}
	resp, err := cl.Get("http://" + addr + "/metrics")
	if err != nil {
		return MetricsSnapshot{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return MetricsSnapshot{}, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return MetricsSnapshot{}, false
	}
	return parseMetrics(string(body)), true
}

// parseMetrics extracts the counters from Prometheus exposition text.
func parseMetrics(text string) MetricsSnapshot {
	var s MetricsSnapshot
	vals := map[string]float64{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name := line
		if i := strings.IndexAny(name, "{ "); i >= 0 {
			name = name[:i]
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			continue
		}
		vals[strings.ToLower(name)] = v
	}
	s.PromptSecondsTotal = metricValue(vals, "prompt_seconds_total", "prompt_eval_seconds", "prompt_seconds")
	s.SlotsActive = int(metricValue(vals, "slots_active", "requests_processing"))
	s.SlotsBusy = int(metricValue(vals, "slots_busy", "n_busy_slots"))
	s.QueueDepth = int(metricValue(vals, "requests_deferred", "queue_depth", "requests_queued"))
	s.ContextTokens = int(metricValue(vals, "kv_cache_tokens", "kv_cache_used_cells"))
	s.KVCacheRatio = metricValue(vals, "kv_cache_usage_ratio", "kv_cache_usage")
	return s
}

// metricValue returns the first matching metric (exact, then substring).
func metricValue(vals map[string]float64, names ...string) float64 {
	for _, n := range names {
		if v, ok := vals[n]; ok {
			return v
		}
	}
	for _, n := range names {
		for k, v := range vals {
			if strings.Contains(k, n) {
				return v
			}
		}
	}
	return 0
}
