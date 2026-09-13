// Package run wraps the subprocess calls the bar makes.
package run

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

// Text captures stdout, discards stderr, and returns "" on any failure.
func Text(timeout time.Duration, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = nil
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// Eww is fire-and-forget with all three streams detached; `eww close` on a
// window that is not open is an expected non-zero.
func Eww(args []string) {
	cmd := exec.Command("eww", args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Run()
}

// Status runs a command for its exit status alone, reporting -1 when the command
// cannot be started at all.
func Status(timeout time.Duration, name string, args ...string) int {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		return -1
	}
	return 0
}
