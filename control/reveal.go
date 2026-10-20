package control

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Reveal opens a file's containing directory in the platform file manager.
// It is intentionally fire-and-forget: the process may live longer than the
// caller and errors are returned only for a failed start.
func Reveal(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	dir := path
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		dir = filepath.Dir(path)
	}
	var cmd string
	switch runtime.GOOS {
	case "windows":
		cmd = "explorer"
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	c := exec.Command(cmd, dir)
	if err := c.Start(); err != nil {
		return err
	}
	go func() { _ = c.Wait() }()
	return nil
}
