package main

import (
	"strings"
	"testing"
)

// TestStartDownloadRejectsUnsafeRefs covers the API/CLI boundary before any
// network call: traversal, bad repo names and a separately supplied unsafe file
// must be refused.
func TestStartDownloadRejectsUnsafeRefs(t *testing.T) {
	a, _ := libApp(t)
	cases := []struct{ ref, file, want string }{
		{"../../etc", "", "invalid model reference"},
		{"owner/name/../secret.gguf", "", "invalid"},
		{"owner/name/%2e%2e/secret.gguf", "", "invalid"},
		{"owner/bad name", "", "invalid model reference"},
		{"owner/name", "../../x.gguf", "invalid file path"},
		{"owner/name", "/abs.gguf", "invalid file path"},
		{"owner/name", "%2e%2e/x.gguf", "invalid file path"},
	}
	for _, c := range cases {
		err := a.StartDownload(c.ref, c.file, "", "")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("StartDownload(%q,%q) = %v, want error containing %q", c.ref, c.file, err, c.want)
		}
	}
	if a.DownloadSnapshot().Active {
		t.Error("a rejected ref must not start a download")
	}
}
