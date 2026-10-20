package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"llamaherd/schema"
)

// TestBuildArgsDraftModel checks that a separate draft sidecar is emitted only
// with a draft-* spec type, and that draft-mtp (embedded heads) never carries
// one.
func TestBuildArgsDraftModel(t *testing.T) {
	joined := func(cfg schema.Config) string { return strings.Join(BuildArgs(cfg), " ") }

	cfg := schema.Config{
		Basic:    schema.Basic{Model: "target.gguf"},
		Advanced: schema.Advanced{SpecType: "draft-simple", DraftModel: "draft.gguf", DraftNMax: 4},
	}
	got := joined(cfg)
	if !strings.Contains(got, "--spec-type draft-simple") {
		t.Fatalf("missing spec type: %s", got)
	}
	if !strings.Contains(got, "--spec-draft-model draft.gguf") {
		t.Fatalf("missing draft model: %s", got)
	}
	if !strings.Contains(got, "--spec-draft-n-max 4") {
		t.Fatalf("missing draft n max: %s", got)
	}

	// No spec type -> the draft model must not leak onto the command line.
	off := joined(schema.Config{
		Basic:    schema.Basic{Model: "target.gguf"},
		Advanced: schema.Advanced{SpecType: "none", DraftModel: "draft.gguf"},
	})
	if strings.Contains(off, "--spec-draft-model") {
		t.Fatalf("draft model emitted with spec type none: %s", off)
	}

	// draft-mtp reads heads from the target; an empty DraftModel adds nothing.
	mtp := joined(schema.Config{
		Basic:    schema.Basic{Model: "target.gguf"},
		Advanced: schema.Advanced{SpecType: "draft-mtp", DraftNMax: 3},
	})
	if !strings.Contains(mtp, "--spec-type draft-mtp") || strings.Contains(mtp, "--spec-draft-model") {
		t.Fatalf("draft-mtp args = %s", mtp)
	}
}

// TestBuildArgsReasoningBudget checks the bonus wiring: 0 = flag omitted,
// non-zero (including -1) is emitted.
func TestBuildArgsReasoningBudget(t *testing.T) {
	joined := func(n int) string {
		return strings.Join(BuildArgs(schema.Config{Basic: schema.Basic{Model: "m.gguf", ReasoningBudget: n}}), " ")
	}
	if got := joined(0); strings.Contains(got, "--reasoning-budget") {
		t.Errorf("0 must omit the flag: %s", got)
	}
	if got := joined(2048); !strings.Contains(got, "--reasoning-budget 2048") {
		t.Errorf("positive budget missing: %s", got)
	}
	if got := joined(-1); !strings.Contains(got, "--reasoning-budget -1") {
		t.Errorf("negative budget missing: %s", got)
	}
}

// TestAwaitHealthTimedOutMarksFailed verifies the new terminal state: a spawn
// that never answers /health becomes "failed" instead of hanging in "starting".
func TestAwaitHealthTimedOutMarksFailed(t *testing.T) {
	old := healthTimeout
	healthTimeout = 50 * time.Millisecond
	defer func() { healthTimeout = old }()

	m := NewManager("/bin/true")
	m.status = StatusStarting
	m.listen = "127.0.0.1:1" // nothing listens here
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.awaitHealth(ctx, "127.0.0.1", 1, time.Now())
	if got := m.Info().Status; got != StatusFailed {
		t.Fatalf("status = %s, want failed", got)
	}
	// A late Stop is still allowed (the process may be alive).
	m.cancel = func() {}
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop on failed = %v", err)
	}
	if got := m.Info().Status; got != StatusStopping {
		t.Fatalf("status after Stop = %s, want stopping", got)
	}
}

func TestParseGenTiming(t *testing.T) {
	gen := "slot print_timings: id  0 | task 5 |        eval time =     520.11 ms /    32 tokens (   16.25 ms per token,    61.53 tokens per second)"
	n, ms, ok := parseGenTiming(gen)
	if !ok || n != 32 || ms != 520.11 {
		t.Fatalf("gen line: got %d %v %v", n, ms, ok)
	}
	prompt := "slot print_timings: id  0 | task 5 | prompt eval time =     120.00 ms /   200 tokens (    0.60 ms per token,  1666.67 tokens per second)"
	if _, _, ok := parseGenTiming(prompt); ok {
		t.Error("prompt eval line must not count as generation speed")
	}
	for _, l := range []string{"", "load_tensors: offloaded 65/65 layers", "       eval time =  10.00 ms /     1 tokens (10 ms per token, 100 tokens per second)"} {
		if _, _, ok := parseGenTiming(l); ok {
			t.Errorf("should not parse %q", l)
		}
	}
}

func TestSpeedStats(t *testing.T) {
	var s SpeedStats
	if s.Summary() != "—" || s.Avg() != 0 {
		t.Fatalf("empty stats: %q", s.Summary())
	}
	s.Add(100, 2000) // 50 tok/s
	s.Add(30, 1000)  // 30 tok/s
	s.Add(60, 1000)  // 60 tok/s (last)
	if s.Min != 30 || s.Max != 60 || s.Last != 60 || s.Requests != 3 {
		t.Errorf("min/max/last/requests = %v %v %v %d", s.Min, s.Max, s.Last, s.Requests)
	}
	// token-weighted: 190 tokens / 4 s
	if got := s.Avg(); got < 47.4 || got > 47.6 {
		t.Errorf("avg = %v, want 47.5", got)
	}
	if !strings.Contains(s.Tooltip(), "min 30.0") || !strings.Contains(s.Tooltip(), "max 60.0") {
		t.Errorf("tooltip = %q", s.Tooltip())
	}
	s.Add(0, 100) // ignored
	if s.Requests != 3 {
		t.Error("zero-token request must be ignored")
	}
}
