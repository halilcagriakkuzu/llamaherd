//go:build unix

package control

import (
	"os/exec"
	"syscall"
)

// terminate asks a running child to exit gracefully (SIGTERM); the exec.Cmd
// WaitDelay escalates to a hard kill if it does not honor the signal.
func terminate(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Signal(syscall.SIGTERM)
}

// signalGraceful sends SIGTERM to a pid.
func signalGraceful(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }

// forceKill sends SIGKILL to a pid.
func forceKill(pid int) error { return syscall.Kill(pid, syscall.SIGKILL) }

// groupAttr starts a child in its own process group so its whole tree can be
// signalled at once (used for the llama.cpp update script).
func groupAttr() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setpgid: true} }

// killGroup signals an entire process group (negative pid).
func killGroup(pid int) { _ = syscall.Kill(-pid, syscall.SIGTERM) }
