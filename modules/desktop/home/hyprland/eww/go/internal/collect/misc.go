package collect

import (
	"fmt"
	"strconv"
	"strings"
)

// The media icons are deliberately inverted: the button shows the action it
// will take, so a PLAYING player gets the pause glyph. Naming them by what they
// depict keeps that visible rather than looking like a transcription slip.
const (
	glyphMemory    = "\uEFC5" // U+EFC5 memory
	glyphPauseIcon = "\uF04C" // U+F04C pause icon -- shown while PLAYING
	glyphPlayIcon  = "\uF04B" // U+F04B play icon -- shown while PAUSED
	glyphStopIcon  = "\uF04D" // U+F04D stop icon
	glyphNoteIcon  = "\uF001" // U+F001 music note -- unknown status
)

// Module is the {text, tooltip, class} shape several collectors return.
type Module struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

// MediaState is the shape collectors.media_state_from_text returns.
type MediaState struct {
	Text    string `json:"text"`
	Status  string `json:"status"`
	Icon    string `json:"icon"`
	Class   string `json:"class"`
	Tooltip string `json:"tooltip"`
}

// SubmapFromEvent mirrors collectors.submap_from_event.
//
// Returns ok=false for the Python None, which the caller distinguishes from the
// empty string: None means "not a submap event, change nothing", "" means "back
// to the default submap, clear the indicator".
func SubmapFromEvent(line string) (string, bool) {
	if !strings.HasPrefix(line, "submap>>") {
		return "", false
	}
	_, value, _ := strings.Cut(line, ">>")
	if value == "default" || value == "reset" {
		return "", true
	}
	return value, true
}

// TrayCountFromText mirrors collectors.tray_count_from_text.
//
// `busctl get-property ... RegisteredStatusNotifierItems` prints the value as
// `as N "item1" "item2" ...` -- the token after the `as` type tag is the count.
func TrayCountFromText(text string) int {
	parts := SplitWhitespaceN(text, -1)
	if len(parts) >= 2 && parts[0] == "as" {
		count, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0
		}
		return count
	}
	return 0
}

// MemoryStateFromText mirrors collectors.memory_state_from_text.
//
// One documented divergence: the original does int(parts[1]) unguarded, so a
// /proc/meminfo line whose second field is not a number raises. This skips it
// instead. /proc/meminfo cannot produce that, and the alternative is a panic in
// a daemon, so the difference is only reachable through a kernel that does not
// exist.
func MemoryStateFromText(text string) Module {
	values := map[string]int64{}
	for _, line := range SplitLines(text) {
		parts := SplitWhitespaceN(line, -1)
		if len(parts) < 2 {
			continue
		}
		number, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		values[strings.TrimRight(parts[0], ":")] = number
	}

	total := values["MemTotal"]
	available := values["MemAvailable"]
	swapTotal := values["SwapTotal"]
	swapFree := values["SwapFree"]
	if total <= 0 {
		return Module{Text: glyphMemory + " --%", Tooltip: "", Class: ""}
	}

	pct := float64(total-available) / float64(total) * 100
	usedG := float64(total-available) / 1048576
	totalG := float64(total) / 1048576
	swapUsedG := float64(swapTotal-swapFree) / 1048576
	swapTotalG := float64(swapTotal) / 1048576
	class := ""
	if pct >= 80 {
		class = "critical"
	}
	return Module{
		Text: fmt.Sprintf("%s %.0f%%", glyphMemory, pct),
		Tooltip: fmt.Sprintf(
			"Used: %.1fG/%.1fG\nSwap: %.1fG/%.1fG", usedG, totalG, swapUsedG, swapTotalG,
		),
		Class: class,
	}
}

// MediaStateFromText mirrors collectors.media_state_from_text.
func MediaStateFromText(statusText, metadataText string) MediaState {
	metadata := SplitLines(metadataText)
	if len(metadata) == 0 || Strip(metadata[0]) == "" {
		return MediaDefault()
	}

	status := ""
	if statusLines := SplitLines(statusText); len(statusLines) > 0 {
		status = Strip(statusLines[0])
	}

	var icon, class, displayStatus string
	switch strings.ToLower(status) {
	case "playing":
		icon, class, displayStatus = glyphPauseIcon, "playing", "Playing"
	case "paused":
		icon, class, displayStatus = glyphPlayIcon, "paused", "Paused"
	case "stopped":
		icon, class, displayStatus = glyphStopIcon, "stopped", "Stopped"
	default:
		icon, class = glyphNoteIcon, ""
		displayStatus = status
		if displayStatus == "" {
			displayStatus = "Media"
		}
	}

	return MediaState{
		Text:   TruncateText(Strip(metadata[0]), 60),
		Status: displayStatus,
		Icon:   icon,
		Class:  class,
		Tooltip: fmt.Sprintf(
			"%s\nLeft click: play/pause\nRight click: next", displayStatus,
		),
	}
}

// MonitorEvent mirrors watchers.monitor_event.
//
// Hyprland emits both v1 and v2 monitor events; matching only the v1 prefixes
// (v2 lines start "monitoraddedv2>>") keeps this single-fire.
func MonitorEvent(line string) (action, name string, ok bool) {
	for _, pair := range [][2]string{
		{"monitoradded>>", "added"},
		{"monitorremoved>>", "removed"},
	} {
		if strings.HasPrefix(line, pair[0]) {
			name := Strip(line[len(pair[0]):])
			if name != "" {
				return pair[1], name, true
			}
		}
	}
	return "", "", false
}

// BarWindowCommand mirrors watchers.bar_window_command.
//
// Must mirror the open-bars script in eww/default.nix so hotplugged monitors
// get the same bar-<name> windows as service startup.
func BarWindowCommand(action, name string) []string {
	if action == "added" {
		return []string{"eww", "open", "bar", "--id", "bar-" + name, "--screen", name, "--arg", "output=" + name}
	}
	return []string{"eww", "close", "bar-" + name}
}

// OpenBarNames mirrors watchers.open_bar_names.
func OpenBarNames(activeWindowsText string) map[string]bool {
	names := map[string]bool{}
	for _, line := range SplitLines(activeWindowsText) {
		windowID, _, _ := strings.Cut(line, ":")
		windowID = Strip(windowID)
		if strings.HasPrefix(windowID, "bar-") {
			names[strings.TrimPrefix(windowID, "bar-")] = true
		}
	}
	return names
}
