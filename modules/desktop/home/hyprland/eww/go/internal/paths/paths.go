// Package paths holds the runtime files the daemon and its clients agree on.
//
// One definition, imported by every side. It exists because there was a period
// when the daemon was Python and the clients were Go, and the socket path was
// duplicated across the boundary and kept in step by hand. The boundary is gone;
// the single definition should outlive the reason for it.
package paths

import (
	"os"
	"path/filepath"
)

// RuntimeFile mirrors paths.runtime_file. The "/tmp" fallback matters: eww-barctl
// can be invoked from a context with no XDG_RUNTIME_DIR, and it has to land on
// the same path the daemon chose.
func RuntimeFile(name string) string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/tmp"
	}
	return filepath.Join(dir, name)
}

// DisplayMode is paths.display_mode_path.
func DisplayMode() string { return RuntimeFile("eww-display-mode") }

// BackendPidfile is paths.backend_pidfile_path.
func BackendPidfile() string { return RuntimeFile("eww-backend.pid") }

// ControlSocket is paths.control_socket_path.
func ControlSocket() string { return RuntimeFile("eww-backend.sock") }
