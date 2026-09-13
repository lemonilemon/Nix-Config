// Package popup drives eww's popup windows for eww-popup.
package popup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"ewwbar/internal/run"
)

// popupWindows is every popup window the helper manages. `eww close` ignores its
// exit status, so naming a window that is already closed is safe.
var popupWindows = []string{
	"volume_popup",
	"bluetooth_popup",
	"network_popup",
	"battery_popup",
	"ai_usage_popup",
	"display_mode_popup",
	"notif_center_popup",
	"wallpaper_picker_popup",
	"settings_popup",
	"calendar_popup",
}

const backdropWindow = "popup_backdrop"

const popupUsage = "usage: eww-popup toggle <window> [screen] | close"

var errPopupUsage = errors.New(popupUsage)

// parseOpenWindows reads `eww active-windows`, one "<id>: <window-name>" per
// line. A popup opened without an explicit --id has id == name.
func parseOpenWindows(activeWindowsText string) map[string]bool {
	openIDs := make(map[string]bool)
	for _, line := range strings.Split(activeWindowsText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		id, _, _ := strings.Cut(line, ":")
		openIDs[strings.TrimSpace(id)] = true
	}
	return openIDs
}

func focusedMonitorFromJSON(monitorsJSON string) string {
	var monitors []map[string]any
	if err := json.Unmarshal([]byte(monitorsJSON), &monitors); err != nil {
		return "0"
	}
	for _, monitor := range monitors {
		if focused, ok := monitor["focused"].(bool); !ok || !focused {
			continue
		}
		if name, ok := monitor["name"].(string); ok && name != "" {
			return name
		}
	}
	return "0"
}

func closeCall() []string {
	return append(append([]string{"close"}, popupWindows...), backdropWindow)
}

func isPopupWindow(name string) bool {
	for _, window := range popupWindows {
		if window == name {
			return true
		}
	}
	return false
}

func popupEwwCalls(command string, args []string, openIDs map[string]bool) ([][]string, error) {
	switch command {
	case "close":
		return [][]string{closeCall()}, nil
	case "toggle":
		if len(args) != 2 {
			return nil, errPopupUsage
		}
		name, screen := args[0], args[1]
		if !isPopupWindow(name) {
			return nil, fmt.Errorf("unknown popup window: %s", name)
		}
		calls := [][]string{closeCall()}
		if !openIDs[name] {
			// --no-daemonize pairs with the open-bars script in eww/default.nix: without it
			// an `eww open` the daemon cannot answer within ~100 ms makes the client fork a
			// second daemon, which opens a second bar.
			calls = append(calls, []string{"--no-daemonize", "open", backdropWindow, "--screen", screen})
			calls = append(calls, []string{"--no-daemonize", "open", name, "--screen", screen})
		}
		return calls, nil
	}
	return nil, errPopupUsage
}

// popupDeps is the seam the tests drive; production wiring is in runPopup.
type popupDeps struct {
	activeWindows func() string
	focusedMon    func() string
	runEww        func([]string)
}

func runPopupWith(argv []string, deps popupDeps) int {
	if len(argv) == 0 || argv[0] == "-h" || argv[0] == "--help" || argv[0] == "help" {
		fmt.Fprintln(os.Stderr, popupUsage)
		return 1
	}
	command, args := argv[0], argv[1:]
	if command == "toggle" && len(args) == 1 {
		args = []string{args[0], deps.focusedMon()}
	}

	openIDs := map[string]bool{}
	if command == "toggle" {
		openIDs = parseOpenWindows(deps.activeWindows())
	}

	calls, err := popupEwwCalls(command, args, openIDs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	for _, call := range calls {
		deps.runEww(call)
	}
	return 0
}

func Run(argv []string) int {
	return runPopupWith(argv, popupDeps{
		activeWindows: func() string { return run.Text(2*time.Second, "eww", "active-windows") },
		focusedMon:    func() string { return focusedMonitorFromJSON(run.Text(2*time.Second, "hyprctl", "monitors", "-j")) },
		runEww:        func(args []string) { run.Eww(args) },
	})
}
