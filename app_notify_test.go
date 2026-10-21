package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"llamaherd/settings"
)

func notificationApp(t *testing.T) (*App, chan string) {
	t.Helper()
	t.Setenv(settings.EnvConfig, filepath.Join(t.TempDir(), "config.json"))
	settings.Reload()
	t.Cleanup(settings.Reload)
	got := make(chan string, 16)
	a := &App{notifier: func(title, body string) error {
		got <- title + "|" + body
		return nil
	}}
	return a, got
}

func waitCalls(t *testing.T, got chan string, n int) []string {
	t.Helper()
	var out []string
	deadline := time.After(2 * time.Second)
	for len(out) < n {
		select {
		case s := <-got:
			out = append(out, s)
		case <-deadline:
			return out
		}
	}
	return out
}

func TestNotifyServerTransitionsOnce(t *testing.T) {
	a, got := notificationApp(t)
	a.notifyServerTransition("stopped", 0)
	a.notifyServerTransition("running", 0)
	a.notifyServerTransition("running", 0)
	calls := waitCalls(t, got, 1)
	if len(calls) != 1 || !strings.Contains(calls[0], "ready") {
		t.Fatalf("calls = %v", calls)
	}
	a.notifyServerTransition("crashed", 3)
	calls = append(calls, waitCalls(t, got, 1)...)
	if len(calls) != 2 || !strings.Contains(calls[1], "crashed") || !strings.Contains(calls[1], "3") {
		t.Fatalf("calls = %v", calls)
	}
}

func TestNotifyEventRespectsToggles(t *testing.T) {
	a, got := notificationApp(t)
	s := settings.NotifySettings()
	s.ServerReady = false
	s.Download = false
	s.Benchmark = false
	s.Enabled = true
	if err := settings.SaveNotificationSettings(s); err != nil {
		t.Fatal(err)
	}
	settings.Reload()
	a.notifyServerTransition("stopped", 0)
	a.notifyServerTransition("running", 0)
	a.notifyEvent("benchmark", "b", "body")
	if calls := waitCalls(t, got, 1); len(calls) != 0 {
		t.Fatalf("disabled classes must stay silent, got %v", calls)
	}
	s.Enabled = false
	if err := settings.SaveNotificationSettings(s); err != nil {
		t.Fatal(err)
	}
	settings.Reload()
	a.notifyEvent("benchmark", "b2", "body")
	if calls := waitCalls(t, got, 1); len(calls) != 0 {
		t.Fatalf("master switch off must silence everything, got %v", calls)
	}
}
