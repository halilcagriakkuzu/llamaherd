package control

import (
	"strings"
	"testing"

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
