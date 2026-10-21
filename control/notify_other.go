//go:build !linux && !darwin && !windows

package control

import "fmt"

// notifyPlatform reports that this platform has no native notifier.
func notifyPlatform(string, string) error {
	return fmt.Errorf("host notifications are not supported on this platform")
}
