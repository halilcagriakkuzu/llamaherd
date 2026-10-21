package doctor

import (
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"llamaherd/settings"
)

func checkAPIPort() Check {
	c := Check{ID: "api-port", Title: "Control-API port"}
	addr := settings.APIAddr()
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		ln.Close()
		c.Status, c.Detail = StatusOK, "free: "+addr
		return c
	}
	if ours(addr) {
		c.Status, c.Detail = StatusOK, "in use by LlamaHerd: "+addr
		return c
	}
	c.Status = StatusFail
	c.Detail = "in use by another process: " + addr
	c.Fix = "choose another port: LLAMAHERD_API_PORT=8518 llamaherd\n" +
		"  or find the holder: ss -ltnp | grep " + settings.APIPort()
	return c
}

// ours probes the control API to tell "already running" from "taken". The
// control API requires its bearer token by default, so the probe sends it too.
func ours(addr string) bool {
	req, err := http.NewRequest(http.MethodGet, "http://"+addr+"/status", nil)
	if err != nil {
		return false
	}
	if tok := settings.APIToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	cl := http.Client{Timeout: 2 * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode == 200 && strings.Contains(string(b), "presetNames")
}
