package collect

import (
	"errors"
	"fmt"
	"time"

	"ewwbar/internal/paths"
)

// Display mode: which monitors are on, and whether the lid and idle inhibitors are
// held. The mode is persisted in a runtime file, so it survives an eww restart but
// not a reboot.

type Display struct {
	Mode         string `json:"mode"`
	LidInhibited string `json:"lid_inhibited"`
	Status       string `json:"status"`
	Class        string `json:"class"`
}

var displayModes = map[string]bool{"normal": true, "external": true, "headless": true}

var modeStatus = map[string]string{
	"normal":   "Normal desktop mode",
	"external": "External display mode",
	"headless": "Headless server mode",
}

// hyprctlTimeout, like systemctlTimeout, exists because these run on a
// control-socket worker with a bounded pool.
const hyprctlTimeout = 5 * time.Second

// ReadDisplayMode treats any unreadable file or unrecognised contents as "normal",
// so a corrupt runtime file cannot wedge the bar in a mode with no monitors on.
func ReadDisplayMode() string {
	text, ok := ReadTextFile(paths.DisplayMode())
	if !ok {
		return "normal"
	}
	if mode := Strip(text); displayModes[mode] {
		return mode
	}
	return "normal"
}

// WriteDisplayMode: "normal" is represented by the file's ABSENCE, not by its
// contents, which is why this deletes rather than writes. Both failure paths are
// swallowed, because raising would abort a switch that has already physically
// happened.
func WriteDisplayMode(mode string) {
	if mode == "normal" {
		_ = RemoveFile(paths.DisplayMode())
		return
	}
	_ = WriteTextFile(paths.DisplayMode(), mode)
}

// DisplayState: an empty status takes the mode's own description.
func DisplayState(status string) Display {
	mode := ReadDisplayMode()
	if status == "" {
		status = modeStatus[mode]
	}
	return Display{
		Mode:         mode,
		LidInhibited: LidInhibitedState(),
		Status:       status,
		Class:        mode,
	}
}

// MonitorState keeps only the JSON objects, which the original did not: a report
// that is not a list of objects raises partway through a display switch. The guard
// cannot simply be "is it a list" -- an EMPTY object survives the original
// untouched, because iterating it yields nothing.
func MonitorState() []map[string]any {
	var monitors []any
	if !ParseJSON(RunText(defaultTimeout, "hyprctl", "monitors", "-j"), &monitors) {
		return nil
	}
	out := []map[string]any{}
	for _, raw := range monitors {
		if monitor, isMap := raw.(map[string]any); isMap {
			out = append(out, monitor)
		}
	}
	return out
}

func runHyprctl(args ...string) error {
	if code := RunStatus(hyprctlTimeout, "hyprctl", args...); code != 0 {
		return fmt.Errorf("hyprctl %s failed", joinArgs(args))
	}
	return nil
}

// Hyprland 0.56 moved hyprctl's control surface to Lua, and the positional forms
// this file used are now syntax errors inside the generated call:
//
//	$ hyprctl dispatch dpms off
//	error: [string "return hl.dispatch(dpms off)"]:1: ')' expected near 'off'
//	exit 7
//
// The `keyword` form was worse, because it fails with exit 0:
//
//	$ hyprctl keyword general:border_size 1
//	keyword can't work with non-legacy parsers. Use eval.
//	exit 0
//
// runHyprctl sees only the code, so external mode would have reported success and
// claimed "External monitors active" with the internal panel still lit. The eval
// form errors with exit 7, which is what makes checking the code meaningful.
const (
	hyprDpmsOn  = `hl.dsp.dpms({ action = "on" })`
	hyprDpmsOff = `hl.dsp.dpms({ action = "off" })`
)

// hyprDisableMonitor is the Lua replacement for `keyword monitor <name>,disable`.
// The field is `disabled`; `disable` and `enabled` are both rejected as unknown.
//
// A partial rule RESETS the fields it omits -- a rule carrying only output and
// disabled reset a live monitor's scale from 1.0 to a default 1.5. Survivable only
// because "normal" runs `hyprctl reload` first, which re-reads the config and the
// per-output overrides in ~/.config/hypr/monitors.lua. Do not optimise that away.
func hyprDisableMonitor(name string) string {
	return fmt.Sprintf("hl.monitor{ output = %q, disabled = true }", name)
}

func joinArgs(args []string) string {
	out := ""
	for i, arg := range args {
		if i > 0 {
			out += " "
		}
		out += arg
	}
	return out
}

// ErrDisplayAction is display.set_display_mode's ValueError for an unknown
// action. Named so the control handler can tell a bad request from a failure.
var ErrDisplayAction = errors.New(
	"display action must be normal, external, headless, restore, toggle, or status")

// ErrNoExternalMonitor is the other ValueError: external mode with nothing to
// switch to would blank every screen.
var ErrNoExternalMonitor = errors.New("external mode requires an active external monitor")

// SetDisplayMode's ORDER of side effects is the contract, not an implementation
// detail:
//
//   - external turns DPMS on BEFORE disabling the internal panel. An "enabled"
//     monitor can still be DPMS-off, and disabling the panel first would leave a
//     black screen with no way back.
//   - normal turns DPMS on and reloads BEFORE dropping the inhibitors.
//   - every path writes the mode file LAST, so a failure part-way leaves the file
//     describing the mode the machine is actually in.
func SetDisplayMode(action string) (Display, error) {
	switch action {
	case "status":
		return DisplayState(""), nil

	case "toggle":
		// Toggle headless on and off. This is the wake path when the screen is
		// already off: the popup's "Turn screens on" button renders on the display
		// that was powered down, so a keybind lets one press sleep and the next cancel.
		if ReadDisplayMode() == "headless" {
			return SetDisplayMode("normal")
		}
		return SetDisplayMode("headless")

	case "restore":
		if err := runHyprctl("dispatch", hyprDpmsOn); err != nil {
			return Display{}, err
		}
		return DisplayState("Screens turned on"), nil

	case "normal":
		if err := runHyprctl("dispatch", hyprDpmsOn); err != nil {
			return Display{}, err
		}
		if err := runHyprctl("reload"); err != nil {
			return Display{}, err
		}
		if _, err := SetLidInhibited(false); err != nil {
			return Display{}, err
		}
		if _, err := SetIdleInhibited(false); err != nil {
			return Display{}, err
		}
		WriteDisplayMode("normal")
		return DisplayState("Exited display mode"), nil

	case "external":
		internal, external := SplitMonitors(MonitorState())
		if len(external) == 0 {
			return Display{}, ErrNoExternalMonitor
		}
		if _, err := SetIdleInhibited(true); err != nil {
			return Display{}, err
		}
		if _, err := SetLidInhibited(true); err != nil {
			return Display{}, err
		}
		if err := runHyprctl("dispatch", hyprDpmsOn); err != nil {
			return Display{}, err
		}
		for _, monitor := range internal {
			name, _ := monitor["name"].(string)
			if name == "" {
				continue
			}
			if err := runHyprctl("eval", hyprDisableMonitor(name)); err != nil {
				return Display{}, err
			}
		}
		WriteDisplayMode("external")
		return DisplayState("External monitors active"), nil

	case "headless":
		if _, err := SetIdleInhibited(true); err != nil {
			return Display{}, err
		}
		if _, err := SetLidInhibited(true); err != nil {
			return Display{}, err
		}
		if err := runHyprctl("dispatch", hyprDpmsOff); err != nil {
			return Display{}, err
		}
		WriteDisplayMode("headless")
		return DisplayState("Headless server mode"), nil
	}
	return Display{}, ErrDisplayAction
}
