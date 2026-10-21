//go:build linux

package control

import (
	"fmt"
	"os/exec"
)

// notifyPlatform uses the freedesktop notification daemon through notify-send.
func notifyPlatform(title, body string) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return fmt.Errorf("no notification backend (install notify-send)")
	}
	return exec.Command("notify-send", "--app-name=LlamaHerd", "--", title, body).Run()
}
