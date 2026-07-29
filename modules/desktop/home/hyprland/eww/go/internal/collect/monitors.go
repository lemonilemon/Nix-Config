package collect

// Which monitor each workspace lives on, and which workspace each monitor is
// showing.
//
// WorkspaceState answers "is workspace N occupied, and is it the focused one
// anywhere". That is all one bar needs on one screen, and it is wrong on two:
// the bar is opened per monitor (defwidget bar [output], one eww window per
// hyprctl monitor) but every copy reads the same workspace_state, so both bars
// highlight the same workspace. Whichever monitor is not focused is therefore
// pointing at a workspace living on the other screen. Verified live with two
// monitors attached: eDP-1 focused on ws1, HDMI-A-1 sitting on ws2,
// ws1_class="active" and ws2_class="occupied", rendered identically on both.
//
// This is deliberately NOT part of state.Bar. The golden replay pins 46
// ControlHandle cases, and each one holds the entire bar state as a byte-exact
// JSON string; a new top-level key breaks all 46, permanently, because the
// recording cannot be regenerated. The daemon publishes this as a plain eww
// variable via `eww update` instead, which the replay never sees.
type MonitorWorkspaces struct {
	// Owner is the monitor each workspace sits on, "" when the workspace does
	// not exist. A workspace exists on exactly one monitor at a time.
	Ws1Owner string `json:"ws1_owner"`
	Ws2Owner string `json:"ws2_owner"`
	Ws3Owner string `json:"ws3_owner"`
	Ws4Owner string `json:"ws4_owner"`
	Ws5Owner string `json:"ws5_owner"`

	// ActiveOn is the monitor currently DISPLAYING that workspace, "" when no
	// monitor is. Distinct from Owner: an unfocused monitor is still showing
	// its own active workspace, which is exactly the case the single shared
	// workspace_state cannot express.
	Ws1ActiveOn string `json:"ws1_active_on"`
	Ws2ActiveOn string `json:"ws2_active_on"`
	Ws3ActiveOn string `json:"ws3_active_on"`
	Ws4ActiveOn string `json:"ws4_active_on"`
	Ws5ActiveOn string `json:"ws5_active_on"`

	// Focused is the monitor with keyboard focus. The bar for this monitor draws
	// its active workspace as the accent; the other bar draws its own active
	// workspace in the dimmer "active elsewhere" treatment.
	Focused string `json:"focused"`

	// Monitors is how many are attached. One means every workspace is owned by
	// the same screen and the yuck can skip the per-monitor treatment entirely,
	// rendering exactly as it does today -- which is what keeps the island from
	// changing shape when the laptop is undocked.
	Monitors int `json:"monitors"`
}

// A flat struct with static keys, matching WorkspaceState next door. The reason
// is the same one recorded there: eww reads each field by name from a simplexpr,
// and indexing an object by a computed key is not something this eww build is
// known to support. Five workspaces because the rest of the bar is five --
// hyprland/default.nix binds MOD1+1..5 and MOD4+1..5, and eww.yuck lists five
// buttons.
const monitorWorkspaceCount = 5

// MonitorWorkspacesFromJSON takes `hyprctl monitors -j` and
// `hyprctl workspaces -j`, rather than running them, so the mapping is testable
// against captured output.
//
// Unparseable input yields the zero value, whose every field is "". The yuck
// falls back to today's rendering when Focused is empty, so a failed read
// degrades to the current behaviour rather than to a blank island.
func MonitorWorkspacesFromJSON(monitorsJSON, workspacesJSON string) MonitorWorkspaces {
	var monitors []any
	if !ParseJSON(monitorsJSON, &monitors) {
		monitors = nil
	}
	var workspaces []any
	if !ParseJSON(workspacesJSON, &workspaces) {
		workspaces = nil
	}

	owner := map[int64]string{}
	for _, raw := range workspaces {
		item, isMap := raw.(map[string]any)
		if !isMap {
			continue
		}
		name, _ := item["monitor"].(string)
		if name == "" {
			continue
		}
		for id := int64(1); id <= monitorWorkspaceCount; id++ {
			if pyEqualsInt(item["id"], id) {
				owner[id] = name
				break
			}
		}
	}

	activeOn := map[int64]string{}
	focused := ""
	attached := 0
	for _, raw := range monitors {
		item, isMap := raw.(map[string]any)
		if !isMap {
			continue
		}
		name, _ := item["name"].(string)
		if name == "" {
			continue
		}
		attached++
		if flag, isBool := item["focused"].(bool); isBool && flag {
			focused = name
		}
		// A monitor with no active workspace is not a shape hyprctl emits, but
		// a missing or non-object activeWorkspace must not take the loop down.
		active, isMap := item["activeWorkspace"].(map[string]any)
		if !isMap {
			continue
		}
		for id := int64(1); id <= monitorWorkspaceCount; id++ {
			if pyEqualsInt(active["id"], id) {
				activeOn[id] = name
				break
			}
		}
	}

	return MonitorWorkspaces{
		Ws1Owner: owner[1], Ws2Owner: owner[2], Ws3Owner: owner[3],
		Ws4Owner: owner[4], Ws5Owner: owner[5],
		Ws1ActiveOn: activeOn[1], Ws2ActiveOn: activeOn[2], Ws3ActiveOn: activeOn[3],
		Ws4ActiveOn: activeOn[4], Ws5ActiveOn: activeOn[5],
		Focused:  focused,
		Monitors: attached,
	}
}
