package control

// validate.go — cheap, side-effect-free form checks used by the snapshot. The
// VRAM estimate is returned alongside so the UI and the Start guard use exactly
// the same numbers.

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"llamaherd/schema"
	"llamaherd/store"
)

// CheckConfig returns human-readable issues for the current form plus the VRAM
// estimate it computed. Severity "error" means Start will fail or be blocked;
// "warn" means it may work but is probably not what the user wants.
func CheckConfig(cfg schema.Config) ([]schema.Issue, VramEstimate) {
	var issues []schema.Issue
	add := func(field, severity, msg string) {
		issues = append(issues, schema.Issue{Field: field, Severity: severity, Message: msg})
	}

	if strings.TrimSpace(cfg.Basic.Model) == "" {
		add("model", "error", "Select a model before starting the server.")
	} else if info, err := store.ReadGGUFInfo(cfg.Basic.Model); err != nil {
		add("model", "warn", "The model file could not be read: "+err.Error())
	} else {
		if cfg.Basic.Context > 0 && info.ContextLength > 0 && cfg.Basic.Context > info.ContextLength {
			add("context", "warn", fmt.Sprintf(
				"Context %d exceeds the model's trained %d tokens.", cfg.Basic.Context, info.ContextLength))
		}
		if n, ok := numericLayers(cfg.Basic.GPULayers); ok && info.BlockCount > 0 && n > info.BlockCount {
			add("gpuLayers", "warn", fmt.Sprintf(
				"-ngl %d exceeds the model's %d layers.", n, info.BlockCount))
		}
	}

	if cfg.Basic.Port <= 0 || cfg.Basic.Port > 65535 {
		add("port", "error", "Port must be between 1 and 65535.")
	}
	if strings.TrimSpace(cfg.Basic.Host) == "" {
		add("host", "warn", "No host set; llama-server will use its default.")
	}
	if cfg.Basic.Context <= 0 {
		add("context", "warn", "No context set; llama-server will use its default.")
	}
	if mm := strings.TrimSpace(cfg.Advanced.MMProj); mm != "" {
		if _, err := os.Stat(mm); err != nil {
			add("mmproj", "warn", "The vision projector file does not exist: "+mm)
		}
	}
	if strings.HasPrefix(cfg.Advanced.SpecType, "draft-") {
		if dm := strings.TrimSpace(cfg.Advanced.DraftModel); dm != "" {
			if _, err := os.Stat(dm); err != nil {
				add("draftModel", "warn", "The draft model file does not exist: "+dm)
			}
		}
	}

	est := EstimateVram(cfg)
	if est.Warn != "" {
		add("context", "error", est.Warn)
	} else if est.TotalVram > 0 && est.GPUPercent >= 88 {
		add("context", "warn", fmt.Sprintf(
			"Estimated VRAM use is %.0f%% of the card; leave headroom for the desktop.", est.GPUPercent))
	}
	return issues, est
}

// numericLayers parses the -ngl field; ok=false for "all"/"auto"/empty.
func numericLayers(v string) (int, bool) {
	s := strings.TrimSpace(strings.ToLower(v))
	if s == "" || s == "all" || s == "auto" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}
