package main

// app_window.go — remembers the main window's geometry. A frameless window has
// no window manager to place or size it, so restoring the last bounds is our
// job; Wails reports screen sizes but not screen origins, so clamping is
// deliberately conservative (size to the largest screen, recentre only when the
// remembered position is implausible).

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"llamaherd/settings"
)

const (
	// windowMinW / windowMinH mirror MinWidth/MinHeight in main.go.
	windowMinW = 960
	windowMinH = 640
	// windowPosLimit is the sanity box for a remembered position; beyond it the
	// old monitor is almost certainly gone.
	windowPosLimit = 10000
)

// windowBounds is the persisted window geometry.
type windowBounds struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Maximised bool `json:"maximised,omitempty"`
}

// windowStatePath is <data dir>/window.json (git-ignored, per user).
func windowStatePath() string { return filepath.Join(settings.DataDir(), "window.json") }

// loadWindowBounds reads the remembered geometry; ok is false when nothing
// usable is stored.
func loadWindowBounds() (windowBounds, bool) {
	b, err := os.ReadFile(windowStatePath())
	if err != nil {
		return windowBounds{}, false
	}
	var w windowBounds
	if err := json.Unmarshal(b, &w); err != nil || w.Width <= 0 || w.Height <= 0 {
		return windowBounds{}, false
	}
	return w, true
}

// saveWindowBounds writes the geometry, creating the data directory if needed.
func saveWindowBounds(w windowBounds) error {
	p := windowStatePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0o644)
}

// clampWindowBounds keeps a remembered window usable: never below the minimum,
// never larger than the biggest screen, and never parked at an implausible
// position (which would leave it off-screen after a monitor change).
func clampWindowBounds(w windowBounds, screens []runtime.Screen) windowBounds {
	if w.Width < windowMinW {
		w.Width = windowMinW
	}
	if w.Height < windowMinH {
		w.Height = windowMinH
	}
	maxW, maxH := 0, 0
	primW, primH := 0, 0
	for _, s := range screens {
		sw, sh := s.Size.Width, s.Size.Height
		if sw <= 0 || sh <= 0 { // older Wails fills only the deprecated fields
			sw, sh = s.Width, s.Height
		}
		if sw > maxW {
			maxW = sw
		}
		if sh > maxH {
			maxH = sh
		}
		if primW == 0 && (s.IsPrimary || len(screens) == 1) {
			primW, primH = sw, sh
		}
	}
	if maxW > 0 && w.Width > maxW {
		w.Width = maxW
	}
	if maxH > 0 && w.Height > maxH {
		w.Height = maxH
	}
	if !plausiblePosition(w) {
		// Centre it on the primary screen; its origin is assumed (0,0) because
		// Wails does not expose per-screen coordinates.
		if primW > 0 {
			w.X = (primW - w.Width) / 2
			w.Y = (primH - w.Height) / 3
		} else {
			w.X, w.Y = 0, 0
		}
	}
	return w
}

// plausiblePosition accepts any coordinate a multi-monitor desktop could
// legitimately use, including negative ones.
func plausiblePosition(w windowBounds) bool {
	return w.X > -windowPosLimit && w.X < windowPosLimit &&
		w.Y > -windowPosLimit && w.Y < windowPosLimit
}

// logWindowMode records the resolved frame in the app log, so `llamaherd logs`
// (and the Config → Logs panel) shows what a desktop-shortcut launch used.
func (a *App) logWindowMode() {
	if settings.Frameless() {
		a.Log("window: frameless (app-drawn title bar)")
		return
	}
	a.Log("window: native frame")
}

// restoreWindowBounds applies the remembered geometry (frameless windows only:
// the native frame delegate already handles this).
func (a *App) restoreWindowBounds() {
	if a.ctx == nil || !settings.Frameless() {
		return
	}
	w, ok := loadWindowBounds()
	if !ok {
		return
	}
	screens, err := runtime.ScreenGetAll(a.ctx)
	if err != nil {
		screens = nil
	}
	w = clampWindowBounds(w, screens)
	// Size first: the window is already mapped at the default size, so only the
	// size can be applied this early without the WM overriding the position.
	runtime.WindowSetSize(a.ctx, w.Width, w.Height)
}

// restoreWindowPlacement runs on DOM-ready, after the window manager has mapped
// the window; applying the position earlier loses a race with the compositor.
//
// WSLg always places the window itself: gtk_window_move is not honoured (each
// launch applies a constant offset, so remembering the reported position would
// walk the window across the screen run after run). Only the size is restored
// there; every other platform restores the position too.
func (a *App) restoreWindowPlacement(ctx context.Context) {
	if !settings.Frameless() {
		return
	}
	w, ok := loadWindowBounds()
	if !ok {
		return
	}
	screens, err := runtime.ScreenGetAll(ctx)
	if err != nil {
		screens = nil
	}
	w = clampWindowBounds(w, screens)
	if !settings.IsWSL() {
		runtime.WindowSetPosition(ctx, w.X, w.Y)
	}
	if w.Maximised {
		runtime.WindowMaximise(ctx)
	}
}

// beforeClose persists the window geometry; returning false lets the close go
// through.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.ctx == nil || !settings.Frameless() {
		return false
	}
	if runtime.WindowIsMaximised(ctx) {
		w, ok := loadWindowBounds()
		if !ok {
			w.Width, w.Height = runtime.WindowGetSize(ctx)
			w.X, w.Y = runtime.WindowGetPosition(ctx)
		}
		// Keep the size the window restores to; only remember "maximised".
		w.Maximised = true
		_ = saveWindowBounds(w)
		return false
	}
	winW, winH := runtime.WindowGetSize(ctx)
	x, y := runtime.WindowGetPosition(ctx)
	_ = saveWindowBounds(windowBounds{Width: winW, Height: winH, X: x, Y: y})
	return false
}
