package doctor

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestParseLlamaVersion(t *testing.T) {
	v, b, c := parseLlamaVersion("version: 0.7.0-dev (build 11708, commit 83f938941)\nbuilt with GNU")
	if v != "0.7.0-dev" || b != 11708 || c != "83f938941" {
		t.Fatalf("got %q %d %q", v, b, c)
	}
	if _, b2, _ := parseLlamaVersion("version: 1 (abc123)"); b2 != 0 {
		t.Fatalf("expected no build number, got %d", b2)
	}
}

func TestSummarize(t *testing.T) {
	if ok, s := summarize([]Check{{Status: StatusOK}, {Status: StatusWarn}}); !ok || !strings.Contains(s, "1 warning") {
		t.Fatalf("ok=%v summary=%q", ok, s)
	}
	if ok, s := summarize([]Check{{Status: StatusFail}}); ok || !strings.Contains(s, "1 failure") {
		t.Fatalf("ok=%v summary=%q", ok, s)
	}
	if ok, s := summarize([]Check{{Status: StatusOK}}); !ok || s != "all checks passed" {
		t.Fatalf("ok=%v summary=%q", ok, s)
	}
}

func TestRunSmoke(t *testing.T) {
	rep := Run()
	if len(rep.Checks) == 0 {
		t.Fatal("no checks returned")
	}
	for _, c := range rep.Checks {
		if c.ID == "" || c.Title == "" {
			t.Fatalf("check missing id/title: %+v", c)
		}
		if c.Status == StatusFail && c.Fix == "" {
			t.Fatalf("failing check %q has no Fix", c.ID)
		}
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

func TestAPIPortFree(t *testing.T) {
	t.Setenv("LLAMAHERD_API_PORT", freePort(t))
	c := checkAPIPort()
	if c.Status != StatusOK {
		t.Fatalf("expected OK on a free port, got %s: %s", c.Status, c.Detail)
	}
}

func TestAPIPortBusy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	t.Setenv("LLAMAHERD_API_PORT", strconv.Itoa(ln.Addr().(*net.TCPAddr).Port))
	if c := checkAPIPort(); c.Status != StatusFail {
		t.Fatalf("expected FAIL on a busy port, got %s: %s", c.Status, c.Detail)
	}
}

func TestAPIPortOursWithToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"status":"running","presetNames":["default"]}`))
	}))
	defer srv.Close()
	t.Setenv("LLAMAHERD_API_TOKEN", "secret")
	addr := strings.TrimPrefix(srv.URL, "http://")
	if !ours(addr) {
		t.Fatal("our own authenticated control API must be recognised")
	}
}
