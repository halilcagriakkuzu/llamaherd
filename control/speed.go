package control

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SpeedStats summarises generation speed (tokens/s) over the requests a managed
// server has finished since it was started. Only servers LlamaHerd launched are
// measured: the numbers come from llama-server's own per-request log line.
type SpeedStats struct {
	Requests int     // completed generations counted
	Tokens   int     // total generated tokens
	TimeMS   float64 // total generation time
	Min      float64 // slowest request, tokens/s
	Max      float64 // fastest request, tokens/s
	Last     float64 // most recent request, tokens/s
}

// Avg is the token-weighted average speed (total tokens / total time), which
// is what a user would measure with a stopwatch; 0 when nothing was recorded.
func (s SpeedStats) Avg() float64 {
	if s.TimeMS <= 0 {
		return 0
	}
	return float64(s.Tokens) / (s.TimeMS / 1000)
}

// Add records one finished generation of n tokens that took ms milliseconds.
func (s *SpeedStats) Add(n int, ms float64) {
	if n <= 0 || ms <= 0 {
		return
	}
	tps := float64(n) / (ms / 1000)
	if s.Requests == 0 || tps < s.Min {
		s.Min = tps
	}
	if tps > s.Max {
		s.Max = tps
	}
	s.Last = tps
	s.Requests++
	s.Tokens += n
	s.TimeMS += ms
}

// Summary is the short value shown in the status box, e.g. "42.3 tok/s".
func (s SpeedStats) Summary() string {
	if s.Requests == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f tok/s", s.Avg())
}

// Tooltip explains the value: min / max / last and how many requests it covers.
func (s SpeedStats) Tooltip() string {
	if s.Requests == 0 {
		return "No generation measured yet (only servers started by LlamaHerd are measured)."
	}
	return fmt.Sprintf("Average of %d request(s), %d tokens\nmin %.1f · max %.1f · last %.1f tok/s",
		s.Requests, s.Tokens, s.Min, s.Max, s.Last)
}

// genTimingRe matches llama-server's per-request generation timing line, e.g.
//
//	"       eval time =     520.11 ms /    32 tokens (   16.25 ms per token,    61.53 tokens per second)"
//
// The prompt-processing variant ("prompt eval time = ...") is excluded by the
// caller.
var genTimingRe = regexp.MustCompile(`\beval time\s*=\s*([0-9.]+)\s*ms\s*/\s*([0-9]+)\s*tokens`)

// parseGenTiming extracts (tokens, ms) from a generation-timing log line.
func parseGenTiming(line string) (tokens int, ms float64, ok bool) {
	i := strings.Index(line, "eval time")
	if i < 0 {
		return 0, 0, false
	}
	// "prompt eval time" is prefill speed, not generation speed.
	if strings.HasSuffix(line[:i], "prompt ") {
		return 0, 0, false
	}
	m := genTimingRe.FindStringSubmatch(line)
	if m == nil {
		return 0, 0, false
	}
	n, err1 := strconv.Atoi(m[2])
	t, err2 := strconv.ParseFloat(m[1], 64)
	if err1 != nil || err2 != nil || n < 2 { // 1-token replies give meaningless tok/s
		return 0, 0, false
	}
	return n, t, true
}

// Speed returns a copy of the current statistics.
func (m *Manager) Speed() SpeedStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.speed
}

// recordLine feeds one server output line to the speed statistics.
func (m *Manager) recordLine(line string) {
	n, ms, ok := parseGenTiming(line)
	if !ok {
		return
	}
	m.mu.Lock()
	m.speed.Add(n, ms)
	m.mu.Unlock()
}
