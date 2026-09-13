// Package paths holds the runtime files the daemon and its clients agree on.
package paths

import (
	"os"
	"path/filepath"
)

// RuntimeFile falls back to /tmp, which eww-barctl relies on when it is invoked
// with no XDG_RUNTIME_DIR and must still land on the daemon's path.
func RuntimeFile(name string) string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/tmp"
	}
	return filepath.Join(dir, name)
}

// StateFile is a file under XDG_STATE_HOME that survives a reboot, falling back
// to ~/.local/state. Empty means no persistence is available.
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

// AiQuotas is the last known provider quota cards.
func AiQuotas() string { return StateFile("ai-quotas.json") }

// Speedtest is the last measured throughput per network, keyed by
// NetworkManager connection UUID.
func Speedtest() string { return StateFile("speedtest.json") }

// NetPolicy is what each network allows -- SSH, DNS integrity, IPv6 -- keyed by
// NetworkManager connection UUID.
func NetPolicy() string { return StateFile("netpolicy.json") }

func DisplayMode() string { return RuntimeFile("eww-display-mode") }

func BackendPidfile() string { return RuntimeFile("eww-backend.pid") }

func ControlSocket() string { return RuntimeFile("eww-backend.sock") }
