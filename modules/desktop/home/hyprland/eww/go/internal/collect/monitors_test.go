package collect

import "testing"

// The exact shapes hyprctl emitted on this machine with both monitors attached,
// trimmed to the keys the collector reads. Captured rather than invented: the
// nesting of activeWorkspace and the fact that an unfocused monitor still
// reports one are the two things this function exists to get right.
const (
	twoMonitorsJSON = `[
      {"name":"eDP-1","focused":false,
       "activeWorkspace":{"id":1,"name":"1"},
       "specialWorkspace":{"id":-98,"name":"special:mainterm"}},
      {"name":"HDMI-A-1","focused":true,
       "activeWorkspace":{"id":2,"name":"2"},
       "specialWorkspace":{"id":0,"name":""}}]`

	twoMonitorWorkspacesJSON = `[
      {"id":1,"name":"1","monitor":"eDP-1","windows":1},
      {"id":-98,"name":"special:mainterm","monitor":"eDP-1","windows":1},
      {"id":2,"name":"2","monitor":"HDMI-A-1","windows":1},
      {"id":3,"name":"3","monitor":"HDMI-A-1","windows":1},
      {"id":4,"name":"4","monitor":"HDMI-A-1","windows":3},
      {"id":5,"name":"5","monitor":"HDMI-A-1","windows":1}]`
)

// TestMonitorWorkspacesSplitsAcrossScreens is the case the bar gets wrong today.
//
// Two monitors, focus on HDMI-A-1. eDP-1 is unfocused but still displaying ws1,
// and the shared workspace_state has no way to say so -- it reports ws1 as the
// one active workspace, which the eDP-1 bar renders correctly by luck and the
// HDMI-A-1 bar renders as a lie.
func TestMonitorWorkspacesSplitsAcrossScreens(t *testing.T) {
	got := MonitorWorkspacesFromJSON(twoMonitorsJSON, twoMonitorWorkspacesJSON)

	want := MonitorWorkspaces{
		Ws1Owner: "eDP-1",
		Ws2Owner: "HDMI-A-1", Ws3Owner: "HDMI-A-1",
		Ws4Owner: "HDMI-A-1", Ws5Owner: "HDMI-A-1",
		// Both monitors display something, and neither entry may be dropped:
		// the unfocused monitor's active workspace is the whole point.
		Ws1ActiveOn: "eDP-1",
		Ws2ActiveOn: "HDMI-A-1",
		Focused:     "HDMI-A-1",
		Monitors:    2,
	}
	if got != want {
		t.Errorf("two-monitor mapping:\n got %+v\nwant %+v", got, want)
	}
}

// TestMonitorWorkspacesUndocked pins the property that keeps the workspace
// island from changing shape when the external screen is unplugged.
//
// One monitor owns everything, so Monitors == 1 and the yuck can take the
// today-path with no per-monitor treatment at all.
func TestMonitorWorkspacesUndocked(t *testing.T) {
	monitors := `[{"name":"eDP-1","focused":true,
	  "activeWorkspace":{"id":3,"name":"3"},
	  "specialWorkspace":{"id":0,"name":""}}]`
	workspaces := `[
	  {"id":1,"name":"1","monitor":"eDP-1","windows":2},
	  {"id":3,"name":"3","monitor":"eDP-1","windows":1}]`

	got := MonitorWorkspacesFromJSON(monitors, workspaces)

	if got.Monitors != 1 {
		t.Errorf("Monitors = %d, want 1", got.Monitors)
	}
	if got.Focused != "eDP-1" {
		t.Errorf("Focused = %q, want %q", got.Focused, "eDP-1")
	}
	if got.Ws3ActiveOn != "eDP-1" {
		t.Errorf("Ws3ActiveOn = %q, want %q", got.Ws3ActiveOn, "eDP-1")
	}
	// A workspace that does not exist yet owns nothing, and must not inherit a
	// neighbour's monitor.
	if got.Ws2Owner != "" || got.Ws4Owner != "" || got.Ws5Owner != "" {
		t.Errorf("absent workspaces claimed an owner: %+v", got)
	}
	if got.Ws1Owner != "eDP-1" {
		t.Errorf("Ws1Owner = %q, want %q", got.Ws1Owner, "eDP-1")
	}
	// ws1 exists but nothing is displaying it.
	if got.Ws1ActiveOn != "" {
		t.Errorf("Ws1ActiveOn = %q, want empty for a workspace no monitor shows",
			got.Ws1ActiveOn)
	}
}

// TestMonitorWorkspacesIgnoresSpecials keeps special:mainterm out of the 1..5
// slots. Its id is negative, so it can never match, but the workspaces list does
// carry it and a change to the matching loop could let it through.
func TestMonitorWorkspacesIgnoresSpecials(t *testing.T) {
	got := MonitorWorkspacesFromJSON(twoMonitorsJSON, twoMonitorWorkspacesJSON)
	for i, owner := range []string{
		got.Ws1Owner, got.Ws2Owner, got.Ws3Owner, got.Ws4Owner, got.Ws5Owner,
	} {
		if owner != "" && owner != "eDP-1" && owner != "HDMI-A-1" {
			t.Errorf("ws%d owner = %q, which is not a monitor name", i+1, owner)
		}
	}
	// special:mainterm lives on eDP-1 with one window; if the id match were
	// loosened it would land in a numbered slot.
	if got.Ws1Owner != "eDP-1" || got.Ws1ActiveOn != "eDP-1" {
		t.Errorf("ws1 disturbed by the special workspace: %+v", got)
	}
}

// TestMonitorWorkspacesDegradesToZero covers every way the read can fail. The
// zero value is load-bearing: the yuck treats an empty Focused as "fall back to
// bar_state.workspace_state", so a failed read renders exactly as the bar does
// today rather than blanking the island.
func TestMonitorWorkspacesDegradesToZero(t *testing.T) {
	cases := []struct {
		name       string
		monitors   string
		workspaces string
	}{
		{"both empty", "", ""},
		{"both malformed", "{oops", "]["},
		// hyprctl timed out or was killed; RunText yields "".
		{"monitors missing", "", twoMonitorWorkspacesJSON},
		// An object where a list belongs.
		{"monitors is an object", `{"name":"eDP-1"}`, twoMonitorWorkspacesJSON},
		// Well-formed JSON whose elements are the wrong type.
		{"elements are scalars", `["eDP-1",7,null]`, `[1,2,3]`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MonitorWorkspacesFromJSON(c.monitors, c.workspaces)
			if got.Focused != "" {
				t.Errorf("Focused = %q, want empty so the yuck falls back",
					got.Focused)
			}
			if got.Monitors != 0 {
				t.Errorf("Monitors = %d, want 0", got.Monitors)
			}
		})
	}
}

// TestMonitorWorkspacesSurvivesAPartialMonitor pins the hardening: one
// unusable entry must not cost the others. A monitor object missing
// activeWorkspace still contributes its name and its focus.
func TestMonitorWorkspacesSurvivesAPartialMonitor(t *testing.T) {
	monitors := `[
	  {"name":"eDP-1","focused":false},
	  {"name":"HDMI-A-1","focused":true,"activeWorkspace":{"id":4,"name":"4"}}]`
	workspaces := `[
	  {"id":4,"name":"4","monitor":"HDMI-A-1","windows":1},
	  {"id":5,"monitor":"","windows":1}]`

	got := MonitorWorkspacesFromJSON(monitors, workspaces)

	if got.Monitors != 2 {
		t.Errorf("Monitors = %d, want 2: a monitor missing activeWorkspace still exists",
			got.Monitors)
	}
	if got.Focused != "HDMI-A-1" {
		t.Errorf("Focused = %q, want HDMI-A-1", got.Focused)
	}
	if got.Ws4ActiveOn != "HDMI-A-1" {
		t.Errorf("Ws4ActiveOn = %q, want HDMI-A-1", got.Ws4ActiveOn)
	}
	// A workspace with an empty monitor name is unattributable, not owned by "".
	if got.Ws5Owner != "" {
		t.Errorf("Ws5Owner = %q, want empty", got.Ws5Owner)
	}
}
