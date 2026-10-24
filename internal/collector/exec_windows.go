//go:build windows

package collector

import (
	"context"
	"os/exec"
	"syscall"
)

// createNoWindow is the CREATE_NO_WINDOW process creation flag.
const createNoWindow = 0x08000000

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// runPowerShell runs a PowerShell script and returns its UTF-8 output.
func runPowerShell(ctx context.Context, script string) (string, error) {
	prelude := "$ProgressPreference='SilentlyContinue';[Console]::OutputEncoding=[Text.Encoding]::UTF8;"
	return runCommand(ctx, "powershell.exe",
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-Command", prelude+script)
}
