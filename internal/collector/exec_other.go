//go:build !windows

package collector

import "os/exec"

func hideWindow(_ *exec.Cmd) {}
