//go:build windows

package control

import (
	"fmt"

	"git.sr.ht/~jackmordaunt/go-toast/v2"
)

// notifyPlatform uses the Windows toast API (the already-vendored go-toast
// dependency; it stubs out to a no-op on other platforms, so it is only built
// here).
func notifyPlatform(title, body string) error {
	n := toast.Notification{AppID: "LlamaHerd", Title: title, Body: body}
	if err := n.Push(); err != nil {
		return fmt.Errorf("windows toast failed: %w", err)
	}
	return nil
}
