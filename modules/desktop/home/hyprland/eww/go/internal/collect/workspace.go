package collect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	glyphWsActive   = "\uF111" // U+F111 filled circle -- active and urgent states
	glyphWsOccupied = "\uF111" // U+F111 filled circle -- same glyph, separate literal upstream
	glyphWsEmpty    = "\uF10C" // U+F10C hollow circle
)

// WorkspaceState is the flat ws1..ws5 class/text map the bar renders.
//
// A struct with explicit fields rather than a map, because encoding/json sorts
// map keys and the original builds these in numeric order. eww.yuck reads each
// by name, so order only matters for byte-compatibility with the Python emit --
// but that is the gate the state emitter has to pass.
type WorkspaceState struct {
	Ws1Class string `json:"ws1_class"`
	Ws1Text  string `json:"ws1_text"`
	Ws2Class string `json:"ws2_class"`
	Ws2Text  string `json:"ws2_text"`
	Ws3Class string `json:"ws3_class"`
	Ws3Text  string `json:"ws3_text"`
	Ws4Class string `json:"ws4_class"`
	Ws4Text  string `json:"ws4_text"`
	Ws5Class string `json:"ws5_class"`
	Ws5Text  string `json:"ws5_text"`
}

// ParseJSON mirrors common.parse_json: decode, or hand back the default on any
// failure. UseNumber throughout so downstream str() stays faithful.
func ParseJSON(text string, into any) bool {
	decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.UseNumber()
	return decoder.Decode(into) == nil
}

// WorkspaceStateFromJSON mirrors collectors.workspace_state_from_json.
func WorkspaceStateFromJSON(activeJSON, workspacesJSON, clientsJSON string) WorkspaceState {
	var active map[string]any
	if !ParseJSON(activeJSON, &active) {
		active = map[string]any{}
	}
	var workspaces []any
	if !ParseJSON(workspacesJSON, &workspaces) {
		workspaces = nil
	}
	var clients []any
	if !ParseJSON(clientsJSON, &clients) {
		clients = nil
	}

	// The raw value, not a coerced int: the original compares
	// `active.get("id", 0) == workspace_id` directly, so the string "2" does
	// NOT match workspace 2 -- but the float 2.0 and the bool True do, because
	// Python numeric equality spans int, float and bool.
	activeID := active["id"]
	if activeID == nil {
		activeID = json.Number("0")
	}

	fields := map[string]string{}
	for id := int64(1); id <= 5; id++ {
		occupied := false
		for _, raw := range workspaces {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if !pyEqualsInt(item["id"], id) {
				continue
			}
			// Only reached once the id matched, mirroring Python's `and`
			// short-circuit. A non-numeric `windows` raises TypeError there;
			// here it reads as not-occupied. Reachable only from a monitor
			// object with an integer id and a string window count, which
			// hyprctl does not emit.
			if windows, ok := toInt(item["windows"]); ok && windows > 0 {
				occupied = true
				break
			}
		}

		urgent := false
		for _, raw := range clients {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			// `is True`, not truthy: a client with urgent=1 does not count.
			if flag, isBool := item["urgent"].(bool); !isBool || !flag {
				continue
			}
			workspace, ok := item["workspace"].(map[string]any)
			if !ok {
				continue
			}
			if pyEqualsInt(workspace["id"], id) {
				urgent = true
				break
			}
		}

		var class, text string
		switch {
		case urgent && pyEqualsInt(activeID, id):
			class, text = "active urgent", glyphWsActive
		case urgent:
			class, text = "occupied urgent", glyphWsActive
		case pyEqualsInt(activeID, id):
			class, text = "active", glyphWsActive
		case occupied:
			class, text = "occupied", glyphWsOccupied
		default:
			class, text = "empty", glyphWsEmpty
		}
		fields[fmt.Sprintf("ws%d_class", id)] = class
		fields[fmt.Sprintf("ws%d_text", id)] = text
	}

	return WorkspaceState{
		Ws1Class: fields["ws1_class"], Ws1Text: fields["ws1_text"],
		Ws2Class: fields["ws2_class"], Ws2Text: fields["ws2_text"],
		Ws3Class: fields["ws3_class"], Ws3Text: fields["ws3_text"],
		Ws4Class: fields["ws4_class"], Ws4Text: fields["ws4_text"],
		Ws5Class: fields["ws5_class"], Ws5Text: fields["ws5_text"],
	}
}

// pyEqualsInt is Python's `value == want` for an int want.
//
// Numbers compare by value across int, float and bool -- 3.0 == 3 and True == 1
// are both True -- while strings, None and containers are never equal to an
// int. toInt must NOT be used here: it parses "3" into 3, which would make a
// string id match a workspace the original leaves empty.
func pyEqualsInt(value any, want int64) bool {
	switch typed := value.(type) {
	case json.Number:
		f, err := typed.Float64()
		return err == nil && f == float64(want)
	case bool:
		if typed {
			return want == 1
		}
		return want == 0
	}
	return false
}

// MissingBarMonitors mirrors watchers.missing_bar_monitors.
func MissingBarMonitors(monitorsJSONText, activeWindowsText string) []string {
	var monitors []any
	if !ParseJSON(monitorsJSONText, &monitors) {
		return []string{}
	}
	open := OpenBarNames(activeWindowsText)
	missing := []string{}
	for _, raw := range monitors {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, ok := item["name"].(string)
		if !ok || name == "" {
			continue
		}
		if !open[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

// ParseAwwwQuery mirrors wallpaper.parse_awww_query.
//
// `awww query` prints one line per output, e.g.
// `eDP-1: 1920x1200, scale: 2, currently displaying: image: /path/img.png`.
// Both monitors mirror, so the first image path wins.
func ParseAwwwQuery(text string) string {
	for _, line := range SplitLines(text) {
		if strings.Contains(line, "image: ") {
			_, rest, _ := strings.Cut(line, "image: ")
			return Strip(rest)
		}
	}
	return ""
}

// InternalMonitorPrefixes mirrors display.INTERNAL_MONITOR_PREFIXES.
var InternalMonitorPrefixes = []string{"eDP-", "LVDS-"}

// IsInternalMonitor mirrors display.is_internal_monitor.
func IsInternalMonitor(name string) bool {
	for _, prefix := range InternalMonitorPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// SplitMonitors mirrors display.split_monitors.
//
// The `disabled` test is `not monitor.get("disabled", False)`, so a monitor
// with no such key counts as enabled and any truthy value disables it.
func SplitMonitors(monitors []map[string]any) (internal, external []map[string]any) {
	internal, external = []map[string]any{}, []map[string]any{}
	for _, monitor := range monitors {
		if pyTruthy(monitor["disabled"]) {
			continue
		}
		name, _ := monitor["name"].(string)
		if IsInternalMonitor(name) {
			internal = append(internal, monitor)
		} else {
			external = append(external, monitor)
		}
	}
	return internal, external
}
