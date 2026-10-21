package main

// app_notify.go — host OS notifications for long-running events. Each class
// can be disabled in settings; when no native backend exists the in-app
// notification center still records the event (the frontend adds every toast).

import (
	"fmt"

	"llamaherd/settings"
)

// notifyEvent sends one host notification when its class is enabled. It never
// blocks the caller: a slow notification daemon must not stall a state update.
func (a *App) notifyEvent(class, title, body string) {
	if !settings.NotifyEvent(class) {
		return
	}
	fn := a.notifier
	if fn == nil {
		return
	}
	a.Log("notify: " + title)
	go func() {
		if err := fn(title, body); err != nil {
			a.Log("notify: " + err.Error())
		}
	}()
}

// notifyServerTransition emits one notification per state transition. The first
// observed state (usually the boot snapshot) is recorded without notifying.
func (a *App) notifyServerTransition(status string, crashExit int) {
	a.notifyMu.Lock()
	prev := a.notifyStatus
	if prev == status {
		a.notifyMu.Unlock()
		return
	}
	a.notifyStatus = status
	a.notifyMu.Unlock()
	if prev == "" {
		return
	}
	switch status {
	case "running":
		a.notifyEvent("ready", "LlamaHerd: server ready", "llama-server is answering requests")
	case "crashed":
		a.notifyEvent("crash", "LlamaHerd: server crashed", fmt.Sprintf("llama-server exited with status %d", crashExit))
	case "failed":
		a.notifyEvent("crash", "LlamaHerd: server not responding", "llama-server has not answered /health")
	}
}

// NotificationSettings returns the effective notification toggles (Wails/API).
func (a *App) NotificationSettings() settings.NotificationSettings {
	return settings.NotifySettings()
}

// SetNotificationSettings persists the toggles and refreshes the snapshot (so
// the UI reflects them immediately).
func (a *App) SetNotificationSettings(s settings.NotificationSettings) error {
	if err := settings.SaveNotificationSettings(s); err != nil {
		return err
	}
	settings.Reload()
	a.Log(fmt.Sprintf("settings: notifications enabled=%v ready=%v crash=%v download=%v benchmark=%v",
		s.Enabled, s.ServerReady, s.ServerCrash, s.Download, s.Benchmark))
	a.emitState()
	return nil
}
