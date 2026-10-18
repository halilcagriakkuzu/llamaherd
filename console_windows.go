//go:build windows

package main

import "syscall"

// hideConsole detaches the console window that Windows attaches to a
// console-subsystem exe, so double-clicking the GUI doesn't leave a terminal
// open. Called only on the GUI path; CLI subcommands keep their console.
func hideConsole() {
	if p, err := syscall.LoadDLL("kernel32.dll"); err == nil {
		if f, err := p.FindProc("FreeConsole"); err == nil {
			f.Call()
		}
	}
}
