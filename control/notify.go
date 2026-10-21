package control

// notify.go — host OS notifications. The platform-specific backends live in
// notify_*.go; app code calls Notify only for user-meaningful events (server
// ready/crash, download finished, benchmark finished), never routine ticks.

import (
	"fmt"
	"strings"
)

// Notify shows one host notification. A missing native backend is not fatal:
// callers log the error and the in-app notification center still has the event.
func Notify(title, body string) error {
	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	if title == "" {
		return fmt.Errorf("notification title required")
	}
	return notifyPlatform(title, body)
}

// escapeAppleScript quotes a string for an AppleScript literal.
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}
