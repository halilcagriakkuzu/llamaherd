package doctor

import (
	"bytes"
	"io"
	"net"
	"net/http"
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
	if owner(addr) == apiOurs {
		c.Status, c.Detail = StatusOK, "in use by LlamaHerd: "+addr
		return c
	}
	c.Status = StatusFail
	c.Detail = "in use by another process: " + addr
	c.Fix = "choose another port: LLAMAHERD_API_PORT=8518 llamaherd\n" +
		"  or find the holder: ss -ltnp | grep " + settings.APIPort()
	return c
}

// apiOwner classifies whatever holds the control-API port.
type apiOwner int

const (
	apiUnknown apiOwner = iota // nothing recognisable: not ours
	apiOurs                    // a LlamaHerd control API
)

// owner asks the port whether it speaks the LlamaHerd control API. It prefers
// the tiny unauthenticated GET /ping beacon and falls back to GET /status for
// builds that predate it.
//
// The old implementation guessed from the first 4 KiB of /status. That was
// wrong: the snapshot starts with the model list, so on any real install
// "presetNames" sat far beyond the read window and our own API was reported as
// "another process".
func owner(addr string) apiOwner {
	code, body, err := probe(addr, "/ping", "")
	if err != nil {
		return apiUnknown
	}
	if code == http.StatusNotFound {
		return ownerFromStatus(addr)
	}
	if code == http.StatusOK && bytes.Contains(body, []byte(`"app":"llamaherd"`)) {
		return apiOurs
	}
	return apiUnknown
}

// ownerFromStatus is the fallback for builds without /ping.
func ownerFromStatus(addr string) apiOwner {
	code, body, err := probe(addr, "/status", settings.APIToken())
	if err != nil || code != http.StatusOK {
		return apiUnknown
	}
	// Read generously: the snapshot leads with the model list, so the marker
	// can be megabytes into the body.
	if bytes.Contains(body, []byte(`"presetNames"`)) {
		return apiOurs
	}
	return apiUnknown
}

// probe issues one GET against the loopback control API. The transport is
// explicitly proxy-free: a configured HTTP(S)_PROXY must never make a local
// listener look like a foreign process.
func probe(addr, path, token string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodGet, "http://"+addr+path, nil)
	if err != nil {
		return 0, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	cl := http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}}
	resp, err := cl.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return resp.StatusCode, b, nil
}
