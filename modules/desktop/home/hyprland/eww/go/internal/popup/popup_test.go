package popup

import (
	"reflect"
	"sort"
	"testing"
)

// These carry forward tests/eww_bar_backend/test_popups.py case for case.

func TestParseOpenWindows(t *testing.T) {
	got := parseOpenWindows("bar-eDP-1: bar\nvolume_popup: volume_popup\npopup_backdrop: popup_backdrop\n")
	want := []string{"bar-eDP-1", "popup_backdrop", "volume_popup"}
	if diff := sortedKeys(got); !reflect.DeepEqual(diff, want) {
		t.Errorf("parseOpenWindows() = %v, want %v", diff, want)
	}
}

func TestParseOpenWindowsIgnoresBlankAndMalformed(t *testing.T) {
	if got := parseOpenWindows("\n   \ngarbage\n"); len(got) != 0 {
		t.Errorf("parseOpenWindows() = %v, want empty", got)
	}
}

func TestToggleClosedOpensBackdropThenPopup(t *testing.T) {
	calls, err := popupEwwCalls("toggle", []string{"volume_popup", "eDP-1"}, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [][]string{
		closeCall(),
		{"open", "popup_backdrop", "--screen", "eDP-1"},
		{"open", "volume_popup", "--screen", "eDP-1"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestToggleAlreadyOpenOnlyCloses(t *testing.T) {
	calls, err := popupEwwCalls("toggle", []string{"volume_popup", "eDP-1"}, map[string]bool{"volume_popup": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(calls, [][]string{closeCall()}) {
		t.Errorf("calls = %v, want a lone close", calls)
	}
}

func TestCloseClosesEverything(t *testing.T) {
	calls, err := popupEwwCalls("close", nil, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(calls, [][]string{closeCall()}) {
		t.Errorf("calls = %v", calls)
	}
	// The close call must name every popup plus the backdrop, or a popup opened
	// on another screen survives the toggle and two end up visible.
	if len(closeCall()) != len(popupWindows)+2 {
		t.Errorf("close call = %v, want close + all popups + the backdrop", closeCall())
	}
}

func TestUnknownWindowRejected(t *testing.T) {
	if _, err := popupEwwCalls("toggle", []string{"nope_popup", "eDP-1"}, map[string]bool{}); err == nil {
		t.Error("unknown popup window should be rejected")
	}
}

func TestToggleRequiresTwoArgs(t *testing.T) {
	if _, err := popupEwwCalls("toggle", []string{"volume_popup"}, map[string]bool{}); err == nil {
		t.Error("toggle with one arg should be rejected at this layer")
	}
}

func TestFocusedMonitorFromJSON(t *testing.T) {
	got := focusedMonitorFromJSON(`[{"name": "eDP-1", "focused": false}, {"name": "HDMI-A-1", "focused": true}]`)
	if got != "HDMI-A-1" {
		t.Errorf("focusedMonitorFromJSON() = %q, want HDMI-A-1", got)
	}
	for _, in := range []string{"", "[]", `[{"name": "eDP-1"}]`, "not json", `{"name":"x"}`} {
		if got := focusedMonitorFromJSON(in); got != "0" {
			t.Errorf("focusedMonitorFromJSON(%q) = %q, want 0", in, got)
		}
	}
}

func TestRunPopupToggleInvokesEwwInOrder(t *testing.T) {
	var seen [][]string
	code := runPopupWith([]string{"toggle", "volume_popup", "HDMI-A-1"}, popupDeps{
		activeWindows: func() string { return "bar-eDP-1: bar\n" },
		focusedMon:    func() string { t.Fatal("focused monitor should not be consulted"); return "" },
		runEww:        func(args []string) { seen = append(seen, args) },
	})
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	want := [][]string{
		closeCall(),
		{"open", "popup_backdrop", "--screen", "HDMI-A-1"},
		{"open", "volume_popup", "--screen", "HDMI-A-1"},
	}
	if !reflect.DeepEqual(seen, want) {
		t.Errorf("eww calls = %v, want %v", seen, want)
	}
}

// eww.yuck invokes `eww-popup toggle display_mode_popup` and
// `... wallpaper_picker_popup` with no screen, so the one-arg path has to
// resolve the focused monitor itself.
func TestToggleWithoutScreenUsesFocusedMonitor(t *testing.T) {
	var seen [][]string
	code := runPopupWith([]string{"toggle", "display_mode_popup"}, popupDeps{
		activeWindows: func() string { return "" },
		focusedMon:    func() string { return "HDMI-A-1" },
		runEww:        func(args []string) { seen = append(seen, args) },
	})
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	want := []string{"open", "display_mode_popup", "--screen", "HDMI-A-1"}
	found := false
	for _, call := range seen {
		if reflect.DeepEqual(call, want) {
			found = true
		}
	}
	if !found {
		t.Errorf("eww calls = %v, want one of them to be %v", seen, want)
	}
}

func TestRunPopupUsage(t *testing.T) {
	for _, argv := range [][]string{nil, {"-h"}, {"--help"}, {"help"}, {"nonsense"}} {
		code := runPopupWith(argv, popupDeps{
			activeWindows: func() string { return "" },
			focusedMon:    func() string { return "0" },
			runEww:        func([]string) { t.Errorf("argv %v should not have run eww", argv) },
		})
		if code != 1 {
			t.Errorf("runPopupWith(%v) = %d, want 1", argv, code)
		}
	}
}

// Every window eww.yuck can toggle must be in popupWindows, or `close` leaves
// it on screen and two popups stack.
func TestEveryYuckPopupIsManaged(t *testing.T) {
	for _, name := range []string{
		"volume_popup", "bluetooth_popup", "network_popup", "battery_popup",
		"ai_usage_popup", "display_mode_popup", "notif_center_popup",
		"wallpaper_picker_popup", "settings_popup",
	} {
		if !isPopupWindow(name) {
			t.Errorf("%s is toggled by eww.yuck but not managed", name)
		}
	}
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
