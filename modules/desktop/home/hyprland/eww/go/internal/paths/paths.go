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

// StateFile is a file under XDG_STATE_HOME that survives a reboot.
//
// Distinct from RuntimeFile, and the distinction is the point: XDG_RUNTIME_DIR
// is tmpfs and is emptied on logout, which is correct for a pidfile and a
// socket and wrong for anything the bar wants to still know tomorrow.
//
// The fallback is ~/.local/state as the XDG base directory spec defines it, not
// /tmp -- landing durable state in /tmp would defeat the reason for the call.
// If even the home directory is unknown the result is empty, which every caller
// reads as "no persistence available" and degrades to in-memory.
func StateFile(name string) string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "eww-bar", name)
}

// AiHistory is the durable daily AI usage rollup behind the popup's History tab.
func AiHistory() string { return StateFile("ai-history.json") }

// AiQuotas is the last known provider quota cards, so the popup opens on real
// numbers instead of "waiting" while the 42 s probe runs.
func AiQuotas() string { return StateFile("ai-quotas.json") }

// DisplayMode is paths.display_mode_path.
func DisplayMode() string { return RuntimeFile("eww-display-mode") }

// BackendPidfile is paths.backend_pidfile_path.
func BackendPidfile() string { return RuntimeFile("eww-backend.pid") }

// ControlSocket is paths.control_socket_path.
func ControlSocket() string { return RuntimeFile("eww-backend.sock") }
