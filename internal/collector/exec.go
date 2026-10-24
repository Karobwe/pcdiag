package collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// defaultCommandTimeout bounds every external command so that a hung tool
// (for example a stuck print spooler) never freezes the application.
const defaultCommandTimeout = 15 * time.Second

// runCommand executes an external program and returns its standard output.
// On Windows the child process is started without a console window.
func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	hideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return stdout.String(), fmt.Errorf("%s: timed out", name)
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return stdout.String(), fmt.Errorf("%s: %w", name, err)
		}
		return stdout.String(), fmt.Errorf("%s: %w: %s", name, err, firstLine(msg))
	}
	return stdout.String(), nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}
