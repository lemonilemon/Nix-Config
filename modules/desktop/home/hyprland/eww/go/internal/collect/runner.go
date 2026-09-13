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
// Package-level vars rather than parameters: a test swaps one and restores it on
// defer instead of threading a runner through forty call sites, and InstallFixture
// drives all of them at once. Nothing in production reassigns these.
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

// defaultTimeout is Python's run_text default. Several call sites override it and
// the differences are load-bearing: playerctl status gets 1 s so a wedged player
// cannot stall the media module, and the startup reconcile gets 5 s.
const defaultTimeout = 2 * time.Second

// The write side of the seams. write_display_mode is the only thing in the backend
// that creates or removes a file, on a path the control socket can be told to
// change, so it goes through a seam like everything else.
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
