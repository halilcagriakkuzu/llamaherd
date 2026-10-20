package main

// app_logs.go — an in-memory log ring plus an optional rotating file. The GUI
// event stream keeps its own timestamp; the file/ring is what survives a crash
// and what `GET /logs`, `llamaherd logs -f` and "Copy diagnostics" read.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"llamaherd/settings"
)

const (
	logRingMax    = 2000
	logFileBackup = 3 // llamaherd.log.1 .. .3
)

// logFileMax is a variable so tests can force rotation with a tiny limit.
var logFileMax int64 = 5 << 20

type logEntry struct {
	Seq  int64
	Text string
}

// recordLog appends one line to the ring and the rotating file. The GUI queue
// is handled by Log itself.
func (a *App) recordLog(line string) {
	text := time.Now().Format("2006-01-02 15:04:05") + " " + line
	a.logMu.Lock()
	defer a.logMu.Unlock()
	a.logSeq++
	a.logRing = append(a.logRing, logEntry{Seq: a.logSeq, Text: text})
	if len(a.logRing) > logRingMax {
		a.logRing = a.logRing[len(a.logRing)-logRingMax:]
	}
	a.writeLogLocked(text + "\n")
}

// writeLogLocked writes to the configured file, opening/rotating as needed.
// The caller holds logMu.
func (a *App) writeLogLocked(line string) {
	path := settings.LogFile()
	if path != a.logPath {
		if a.logFile != nil {
			_ = a.logFile.Close()
			a.logFile = nil
		}
		a.logPath = path
		a.logBytes = 0
		a.logFailed = false
	}
	if path == "" {
		return
	}
	if !a.openLogLocked(path) {
		return
	}
	if a.logBytes+int64(len(line)) > logFileMax {
		a.rotateLogLocked()
		if !a.openLogLocked(path) {
			return
		}
	}
	n, err := a.logFile.WriteString(line)
	a.logBytes += int64(n)
	if err != nil {
		a.logWriteErrorLocked(err)
	}
}

// openLogLocked opens the active file when needed and reports success.
func (a *App) openLogLocked(path string) bool {
	if a.logFile != nil {
		return true
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		a.logWriteErrorLocked(err)
		return false
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		a.logWriteErrorLocked(err)
		return false
	}
	if st, err := f.Stat(); err == nil {
		a.logBytes = st.Size()
	}
	a.logFile = f
	return true
}

func (a *App) rotateLogLocked() {
	path := a.logPath
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
	_ = os.Remove(path + "." + strconv.Itoa(logFileBackup))
	for i := logFileBackup; i > 1; i-- {
		_ = os.Rename(path+"."+strconv.Itoa(i-1), path+"."+strconv.Itoa(i))
	}
	_ = os.Rename(path, path+".1")
	a.logBytes = 0
}

// logWriteErrorLocked reports a file problem once per path, then keeps the
// in-memory ring working.
func (a *App) logWriteErrorLocked(err error) {
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
	if !a.logFailed {
		a.logFailed = true
		fmt.Fprintf(os.Stderr, "llamaherd: log file disabled: %v\n", err)
	}
}

// resetLog closes the current file so the next line uses the (possibly new)
// settings path.
func (a *App) resetLog() {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
	a.logPath = ""
	a.logBytes = 0
	a.logFailed = false
}

// LogTail returns the last n ring lines (n <= 0 means all), plus the sequence
// number of the newest line.
func (a *App) LogTail(n int) ([]string, int64) {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	if n <= 0 || n > len(a.logRing) {
		n = len(a.logRing)
	}
	out := make([]string, 0, n)
	for _, e := range a.logRing[len(a.logRing)-n:] {
		out = append(out, e.Text)
	}
	return out, a.logSeq
}

// LogsAfter returns ring lines newer than seq (for `logs -f`).
func (a *App) LogsAfter(seq int64) ([]string, int64) {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	out := make([]string, 0, 32)
	for _, e := range a.logRing {
		if e.Seq > seq {
			out = append(out, e.Text)
		}
	}
	return out, a.logSeq
}

// LogsJSON is the control API representation of the log ring.
func (a *App) LogsJSON(tail int, after int64) ([]byte, error) {
	lines, next := a.LogTail(tail)
	if after >= 0 {
		lines, next = a.LogsAfter(after)
	}
	a.logMu.Lock()
	path, enabled := a.logPath, a.logPath != ""
	a.logMu.Unlock()
	if path == "" {
		path = settings.LogFile()
		enabled = path != ""
	}
	return json.Marshal(map[string]any{
		"lines":   lines,
		"next":    next,
		"path":    path,
		"enabled": enabled,
	})
}

// DiagnosticsText is the "copy this into an issue" blob: versions, paths,
// environment report and the recent log tail.
func (a *App) DiagnosticsText() (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "LlamaHerd %s\n%s/%s\n\n", version, runtime.GOOS, runtime.GOARCH)
	a.stateMu.RLock()
	fmt.Fprintf(&b, "server bin: %s\nmodels dir: %s\nlog file: %s\n\n",
		a.bin, a.modelsDir, settings.LogFile())
	a.stateMu.RUnlock()
	rep := a.DoctorReport()
	fmt.Fprintf(&b, "doctor: %s (ok=%v)\n", rep.Summary, rep.OK)
	for _, c := range rep.Checks {
		fmt.Fprintf(&b, "[%s] %s", c.Status, c.Title)
		if c.Detail != "" {
			fmt.Fprintf(&b, " — %s", c.Detail)
		}
		b.WriteString("\n")
	}
	lines, _ := a.LogTail(200)
	b.WriteString("\n--- last log lines ---\n")
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n")
	return b.String(), nil
}
