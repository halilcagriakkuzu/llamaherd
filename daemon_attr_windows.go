//go:build windows

package main

import "syscall"

// daemonSysProcAttr detaches the spawned headless daemon from the console so it
// survives the launching terminal (DETACHED_PROCESS).
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000008}
}
