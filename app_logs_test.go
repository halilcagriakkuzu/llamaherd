package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"llamaherd/settings"
)

func TestLogRingAndFile(t *testing.T) {
	a, _ := libApp(t)
	a.Log("first line")
	a.Log("second line")

	lines, seq := a.LogTail(1)
	if len(lines) != 1 || !strings.Contains(lines[0], "second line") {
		t.Fatalf("LogTail = %v (seq %d)", lines, seq)
	}
	if _, next := a.LogTail(0); next != seq {
		t.Errorf("tail(0) seq = %d, want %d", next, seq)
	}
	after, _ := a.LogsAfter(seq - 1)
	if len(after) != 1 || !strings.Contains(after[0], "second line") {
		t.Errorf("LogsAfter = %v", after)
	}

	path := settings.LogFile()
	if path == "" {
		t.Fatal("expected file logging to be enabled")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "first line") || !strings.Contains(string(data), "second line") {
		t.Errorf("log file missing lines: %q", string(data))
	}

	raw, err := a.LogsJSON(10, -1)
	if err != nil {
		t.Fatal(err)
	}
	var resp struct {
		Lines   []string `json:"lines"`
		Next    int64    `json:"next"`
		Path    string   `json:"path"`
		Enabled bool     `json:"enabled"`
	}
	if json.Unmarshal(raw, &resp) != nil || len(resp.Lines) != 2 || !resp.Enabled || resp.Next != seq {
		t.Errorf("LogsJSON = %s", raw)
	}
}

func TestLogFileRotation(t *testing.T) {
	old := logFileMax
	logFileMax = 120
	defer func() { logFileMax = old }()

	a, _ := libApp(t)
	for i := 0; i < 20; i++ {
		a.Log("a reasonably long log line to force rotation")
	}
	path := settings.LogFile()
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected a rotated %s.1: %v", path, err)
	}
	if st, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if st.Size() > logFileMax {
		t.Errorf("active log is %d bytes, max %d", st.Size(), logFileMax)
	}
}

func TestLogFileOptOut(t *testing.T) {
	a, root := libApp(t)
	t.Setenv(settings.EnvLogFile, "off")
	settings.Reload()
	a.resetLog()
	a.Log("not written")
	if data, err := os.ReadFile(settings.LogFile()); err == nil && strings.Contains(string(data), "not written") {
		t.Errorf("log written despite opt-out: %s", data)
	}
	_ = root
}
