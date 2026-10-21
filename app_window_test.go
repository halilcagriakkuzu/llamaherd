package main

import (
	"testing"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func TestClampWindowBoundsMinimum(t *testing.T) {
	got := clampWindowBounds(windowBounds{Width: 300, Height: 200}, nil)
	if got.Width != windowMinW || got.Height != windowMinH {
		t.Fatalf("small bounds = %dx%d, want %dx%d", got.Width, got.Height, windowMinW, windowMinH)
	}
}

func TestClampWindowBoundsScreenSize(t *testing.T) {
	screens := []runtime.Screen{
		{IsPrimary: true, IsCurrent: true, Width: 1366, Height: 768},
		{Width: 1024, Height: 600},
	}
	got := clampWindowBounds(windowBounds{Width: 2400, Height: 1400, X: 10, Y: 10}, screens)
	if got.Width != 1366 || got.Height != 768 {
		t.Fatalf("bounds = %dx%d, want the largest screen 1366x768", got.Width, got.Height)
	}
	if got.X != 10 || got.Y != 10 {
		t.Fatalf("plausible position moved to %d,%d", got.X, got.Y)
	}
}

func TestClampWindowBoundsRecentresStalePosition(t *testing.T) {
	screens := []runtime.Screen{{IsPrimary: true, Width: 1920, Height: 1080}}
	got := clampWindowBounds(windowBounds{Width: 1240, Height: 820, X: 20000, Y: -30000}, screens)
	if got.X != (1920-1240)/2 || got.Y != (1080-820)/3 {
		t.Fatalf("stale position = %d,%d, want it centred on the primary screen", got.X, got.Y)
	}
}

func TestClampWindowBoundsLegacySizeFields(t *testing.T) {
	// Older Wails builds fill only the deprecated Width/Height fields.
	screens := []runtime.Screen{{IsPrimary: true, Width: 1280, Height: 720}}
	got := clampWindowBounds(windowBounds{Width: 4000, Height: 4000}, screens)
	if got.Width != 1280 || got.Height != 720 {
		t.Fatalf("bounds = %dx%d, want 1280x720", got.Width, got.Height)
	}
}

func TestWindowBoundsRoundTrip(t *testing.T) {
	t.Setenv("LLAMAHERD_DATA_DIR", t.TempDir())
	want := windowBounds{Width: 1400, Height: 900, X: 40, Y: 60, Maximised: true}
	if err := saveWindowBounds(want); err != nil {
		t.Fatal(err)
	}
	got, ok := loadWindowBounds()
	if !ok {
		t.Fatal("saved bounds did not load")
	}
	if got != want {
		t.Fatalf("bounds = %+v, want %+v", got, want)
	}
}

func TestLoadWindowBoundsMissing(t *testing.T) {
	t.Setenv("LLAMAHERD_DATA_DIR", t.TempDir())
	if _, ok := loadWindowBounds(); ok {
		t.Fatal("missing file must not report bounds")
	}
}
