package collect

import (
	"os"
	"time"

	"ewwbar/internal/run"
)

// RunText and ReadTextFile are the seams every impure collector goes through.
//
// Package-level vars rather than parameters or an interface, deliberately: the
// Python side is tested with ~25 unittest.mock.patch calls against run_text,
// and this shape lets the ported tests keep the same structure instead of
// threading a runner through forty call sites. Reassign, defer the restore.
//
// Nothing in production reassigns these. If something ever needs to, it wants a
// parameter instead.
var (
	RunText = func(timeout time.Duration, name string, args ...string) string {
		return run.Text(timeout, name, args...)
	}

	ReadTextFile = func(path string) (string, bool) {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", false
		}
		return string(data), true
	}
)

// Python's run_text default. Spelled out because several call sites override it
// and the differences are load-bearing: playerctl status gets 1 s so a wedged
// player cannot stall the media module, and the startup reconcile gets 5 s
// because hyprctl and eww are both slow right after a config reload.
const defaultTimeout = 2 * time.Second
