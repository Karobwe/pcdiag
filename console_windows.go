//go:build windows

package main

import (
	"os"
	"syscall"
)

// attachConsole reconnects the standard streams to the parent console.
// Release builds use the "windowsgui" subsystem, which has no console, so
// without this "pcdiag -report text" would print nothing in cmd.exe or
// PowerShell. Redirected streams (pipes, files) are left untouched.
func attachConsole() {
	if h, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE); err == nil && h != 0 && h != syscall.InvalidHandle {
		return
	}
	const attachParentProcess = ^uintptr(0) // (DWORD)-1
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("AttachConsole")
	if r, _, _ := proc.Call(attachParentProcess); r == 0 {
		return
	}
	if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout = f
		os.Stderr = f
	}
}
