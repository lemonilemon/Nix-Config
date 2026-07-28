package collect

import (
	"errors"
	"os"
	"path/filepath"
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

// The write side of the seams. display.write_display_mode is the only thing in
// the backend that creates or removes a file, and it does so on a path the
// control socket can be told to change -- so it goes through a seam like
// everything else, and the equivalence gate can assert what it wrote without
// touching the real runtime directory.
var (
	RunStatus = func(timeout time.Duration, name string, args ...string) int {
		return run.Status(timeout, name, args...)
	}

	WriteTextFile = func(path, text string) error {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(text), 0o644)
	}

	RemoveFile = func(path string) error {
		err := os.Remove(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
)
