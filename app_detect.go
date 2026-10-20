package main

// app_detect.go — first-run/discovery helper: find a working llama-server
// without asking the user to type an absolute path.

import (
	"fmt"
	"runtime"
	"strings"

	"llamaherd/control"
	"llamaherd/settings"
)

// DetectedServer is the result of DetectServerBin.
type DetectedServer struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

// DetectServerBin probes the current setting, PATH and the usual llama.cpp
// build locations (plus WSL distros on Windows) and returns the first candidate
// that answers `--version`. It never changes the saved path; the caller decides.
func (a *App) DetectServerBin() (DetectedServer, error) {
	a.stateMu.RLock()
	current := a.bin
	a.stateMu.RUnlock()

	var candidates []string
	candidates = append(candidates, current, settings.ServerBin())
	candidates = append(candidates, settings.ServerBinCandidates()...)
	if runtime.GOOS == "windows" {
		candidates = append(candidates, control.DiscoverWSLBinaries()...)
	}

	seen := map[string]bool{}
	var lastErr error
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		if !control.BinaryExists(c) {
			continue
		}
		v, err := control.ProbeBinary(c)
		if err != nil {
			lastErr = err
			continue
		}
		a.Log(fmt.Sprintf("detect: llama-server %s (%s)", c, v))
		return DetectedServer{Path: c, Version: v}, nil
	}
	if lastErr != nil {
		return DetectedServer{}, fmt.Errorf("found a llama-server but it did not run: %w", lastErr)
	}
	return DetectedServer{}, fmt.Errorf("no llama-server found in PATH or the usual llama.cpp build locations")
}
