//go:build !windows

package collector

import (
	"context"
	"os/exec"
	"strings"
)

func collectCUPS(ctx context.Context) (PrintingInfo, error) {
	info := PrintingInfo{Printers: []Printer{}}
	if _, err := exec.LookPath("lpstat"); err != nil {
		info.SpoolerStatus = "Not installed"
		return info, nil
	}
	if out, err := runCommand(ctx, "lpstat", "-r"); err == nil && strings.Contains(out, "not running") {
		info.SpoolerStatus = "Stopped"
		return info, nil
	}
	info.SpoolerStatus = "Running"

	printers, err := runCommand(ctx, "lpstat", "-p", "-d")
	if err != nil && printers == "" {
		// lpstat exits non-zero when no printer is installed.
		if strings.Contains(err.Error(), "No destinations") {
			return info, nil
		}
		return info, err
	}
	devices, _ := runCommand(ctx, "lpstat", "-v")
	info.Printers = parseLpstat(printers, devices)
	return info, nil
}
