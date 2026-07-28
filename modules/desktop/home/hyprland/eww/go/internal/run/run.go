// Package run wraps the subprocess calls the bar makes.
package run

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

// Text mirrors common.run_text: capture stdout, discard stderr, and return
// the empty string on any failure -- timeout, non-zero exit, or missing binary.
// Callers parse the result and fall back to a default, so an error string here
// would only be re-flattened to "".
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

// Eww mirrors popups._run_eww: fire-and-forget, all three streams detached,
// exit status ignored. `eww close` on a window that is not open is a normal,
// expected non-zero.
func Eww(args []string) {
	cmd := exec.Command("eww", args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Run()
}

// Status runs a command for its exit status alone, with all three streams
// detached. Mirrors the subprocess.run(..., check=False) calls in inhibitors
// and display, which look only at returncode.
//
// A command that cannot be started at all reports -1 where CPython raises
// FileNotFoundError. Deliberate: the callers read this as "the service is not
// active" or "the hyprctl failed", and a missing systemctl should degrade the
// bar rather than kill the daemon thread.
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
