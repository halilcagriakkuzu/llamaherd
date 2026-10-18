package doctor

import (
	"runtime"
	"strings"
)

const webview2GUID = "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"

func webview2Check(c Check) Check {
	if runtime.GOOS != "windows" {
		c.Status, c.Detail = StatusSkip, "not applicable"
		return c
	}
	keys := []string{
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + webview2GUID,
		`HKLM\SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webview2GUID,
		`HKCU\SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webview2GUID,
	}
	for _, k := range keys {
		if out, err := execOut("reg", "query", k, "/v", "pv"); err == nil {
			if f := strings.Fields(out); len(f) >= 3 {
				c.Status, c.Detail = StatusOK, "WebView2 runtime "+f[len(f)-1]
				return c
			}
		}
	}
	c.Status = StatusWarn
	c.Detail = "WebView2 runtime not detected"
	c.Fix = "Install the Evergreen WebView2 Runtime (developer.microsoft.com/microsoft-edge/webview2)"
	return c
}
