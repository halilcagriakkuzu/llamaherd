package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusSummary(t *testing.T) {
	raw := []byte(`{
		"status":"running","alias":"qwen","dirty":true,
		"presetNames":["default","high-ctx"],"selectedPreset":1,
		"server":{"listen":"127.0.0.1:8090","context":"32768","gpuLayers":"all","tokPerSec":"47.5 tok/s","vram":"14.9 GB","uptime":"00:10:00"},
		"download":{"active":true,"file":"model.gguf","percent":43}
	}`)
	got, err := statusSummary(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"status:  running",
		"model:   qwen [high-ctx] (unsaved changes)",
		"127.0.0.1:8090",
		"ctx 32768",
		"ngl all",
		"47.5 tok/s",
		"VRAM 14.9 GB",
		"up 00:10:00",
		"download: 43% model.gguf",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("summary missing %q:\n%s", want, got)
		}
	}
}

func TestParseImportArgs(t *testing.T) {
	req, check, err := parseImportArgs([]string{"a.gguf", "--move", "dir", "--mmproj", "mm.gguf", "--allow-duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	paths := req["paths"].([]string)
	if check || len(paths) != 2 || !filepath.IsAbs(paths[0]) || !strings.HasSuffix(paths[1], "dir") {
		t.Errorf("paths = %v check = %v", paths, check)
	}
	if req["move"] != true || req["allowDuplicate"] != true || !filepath.IsAbs(req["mmproj"].(string)) {
		t.Errorf("req = %+v", req)
	}
	if _, check, _ = parseImportArgs([]string{"--check", "x.gguf"}); !check {
		t.Error("--check not parsed")
	}
	for _, bad := range [][]string{{}, {"--move"}, {"x.gguf", "--mmproj"}, {"x.gguf", "--bogus"}} {
		if _, _, err := parseImportArgs(bad); err == nil {
			t.Errorf("args %v accepted", bad)
		}
	}
}

func TestImportReportSummary(t *testing.T) {
	raw := []byte(`{"ok":true,"report":{"items":[
		{"source":"/in/a.gguf","status":"imported","alias":"a"},
		{"source":"/in/b.gguf","status":"duplicate","message":"already exists as b (/m/b.gguf)"},
		{"source":"/in/c.gguf","status":"error","message":"boom"}]}}`)
	out, status, err := importReportSummary(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{"imported   a", "skipped    /in/b.gguf", "error      /in/c.gguf: boom"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary missing %q:\n%s", want, got)
		}
	}
	if status == 0 {
		t.Error("a failed item must give a non-zero status")
	}
	if _, status, _ := importReportSummary([]byte(`{"ok":true,"report":{"items":[{"source":"x","status":"imported","alias":"x"}]}}`)); status != 0 {
		t.Errorf("clean import status = %d", status)
	}
}
