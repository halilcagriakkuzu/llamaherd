//go:build darwin

package control

import (
	"fmt"
	"os/exec"
	"strings"
)

// escapeAppleScript quotes a string for an AppleScript literal.
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}

// notifyPlatform uses AppleScript's display notification.
func notifyPlatform(title, body string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, escapeAppleScript(body), escapeAppleScript(title))
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		return fmt.Errorf("osascript notification failed: %w", err)
	}
	return nil
}
