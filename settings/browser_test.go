package settings

import "testing"

func TestOpenExternalURLRejectsEmpty(t *testing.T) {
	if err := OpenExternalURL("   "); err == nil {
		t.Fatal("empty url must be rejected")
	}
}

func TestPSQuote(t *testing.T) {
	if got, want := psQuote("it's fine"), "it''s fine"; got != want {
		t.Fatalf("psQuote = %q, want %q", got, want)
	}
}

func TestOpenExternalURLReportsFailure(t *testing.T) {
	// An empty PATH means no launcher can start: the caller must get an error
	// instead of a silent no-op.
	t.Setenv("PATH", t.TempDir())
	if err := OpenExternalURL("http://127.0.0.1:9/"); err == nil {
		t.Fatal("expected an error when no browser launcher is available")
	}
}
