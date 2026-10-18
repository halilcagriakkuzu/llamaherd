//go:build unix

package main

import "syscall"

// daemonSysProcAttr detaches the spawned headless daemon from this process: a
// new session so it survives the launching terminal.
func daemonSysProcAttr() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }
