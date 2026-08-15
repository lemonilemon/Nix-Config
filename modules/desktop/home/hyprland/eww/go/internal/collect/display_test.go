package collect

import (
	"strings"
	"testing"

	"ewwbar/internal/paths"
)

// These replace the 13 recorded cases that internal/replay steps over. Those
// cases pin hyprctl's pre-0.56 command line, which Hyprland no longer accepts;
// see recordsSupersededHyprctl for why the recording is skipped rather than
// edited. What follows asserts the same behaviour against the surface that
// exists now.

// journalHas reports whether the fixture journal contains an entry, and returns
// its index so ordering can be asserted.
func journalIndex(t *testing.T, want string) int {
	t.Helper()
	for i, entry := range FixtureJournal() {
		if entry == want {
			return i
		}
	}
	t.Errorf("journal has no %q\n  journal: %s", want,
		strings.Join(FixtureJournal(), "\n           "))
	return -1
}

const (
	dpmsOnCall  = "status\x1fhyprctl\x1fdispatch\x1f" + hyprDpmsOn
	dpmsOffCall = "status\x1fhyprctl\x1fdispatch\x1f" + hyprDpmsOff
)

// The bug this whole file exists for. `hyprctl dispatch dpms off` is a syntax
// error under Hyprland 0.56 and exits 7, so SetDisplayMode returned before
// WriteDisplayMode and the mode file was never created -- which meant
// ReadDisplayMode kept answering "normal" and the panel kept saying "Normal
// desktop" however many times the button was pressed. Nothing reported it,
// because every button passes --quiet.
func TestHeadlessUsesTheLuaDpmsCallAndRecordsTheMode(t *testing.T) {
	restore := InstallFixture(map[string]string{"env.XDG_RUNTIME_DIR": "/run"})
	defer restore()

	if _, err := SetDisplayMode("headless"); err != nil {
		t.Fatalf("headless: %v", err)
	}

	dpms := journalIndex(t, dpmsOffCall)
	// The mode file is the thing that was never written. Asserted through the
	// journal rather than the returned Display, because the fixture's
	// WriteTextFile only records the call -- it does not store the value, so a
	// later ReadTextFile cannot see it.
	wrote := journalIndex(t, "write\x1f"+paths.DisplayMode()+"\x1fheadless")

	// Written LAST, so a failure part-way leaves the file describing the mode
	// the machine is actually in.
	if dpms >= 0 && wrote >= 0 && wrote < dpms {
		t.Errorf("mode file written before the screens went off (%d < %d)", wrote, dpms)
	}
}

func TestNormalUsesTheLuaDpmsCall(t *testing.T) {
	restore := InstallFixture(map[string]string{
		"file\x1f" + "/run/eww-display-mode": "headless",
		"env.XDG_RUNTIME_DIR":                "/run",
	})
	defer restore()

	if _, err := SetDisplayMode("normal"); err != nil {
		t.Fatalf("normal: %v", err)
	}

	dpms := journalIndex(t, dpmsOnCall)
	reload := journalIndex(t, "status\x1fhyprctl\x1freload")

	// The screens have to be back before hypridle is allowed to act again, so
	// the reload precedes the inhibitor release. Asserting the pair keeps that
	// ordering from being reshuffled by a later edit.
	if dpms >= 0 && reload >= 0 && dpms > reload {
		t.Errorf("dpms on ran after reload (%d > %d)", dpms, reload)
	}
}

func TestRestoreUsesTheLuaDpmsCall(t *testing.T) {
	restore := InstallFixture(map[string]string{})
	defer restore()

	display, err := SetDisplayMode("restore")
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	journalIndex(t, dpmsOnCall)
	if display.Status != "Screens turned on" {
		t.Errorf("status = %q", display.Status)
	}
}

// External mode's other call was the dangerous one. `hyprctl keyword monitor
// <name>,disable` answers "keyword can't work with non-legacy parsers" and
// exits ZERO, so runHyprctl saw success: the mode file would be written and the
// panel would report "External monitors active" with the internal display still
// lit. The eval form errors with exit 7 like everything else.
func TestExternalDisablesTheInternalPanelThroughEval(t *testing.T) {
	restore := InstallFixture(map[string]string{
		"hyprctl\x1fmonitors\x1f-j": `[
		  {"name":"eDP-1","disabled":false},
		  {"name":"DP-3","disabled":false}
		]`,
		"env.XDG_RUNTIME_DIR": "/run",
	})
	defer restore()

	if _, err := SetDisplayMode("external"); err != nil {
		t.Fatalf("external: %v", err)
	}

	disable := journalIndex(t,
		"status\x1fhyprctl\x1feval\x1f"+hyprDisableMonitor("eDP-1"))
	dpms := journalIndex(t, dpmsOnCall)

	// DPMS on BEFORE disabling the panel: an enabled monitor can still be
	// DPMS-off, arriving here from headless or an idle blank, and disabling it
	// first would leave a dark screen with no way back.
	if dpms >= 0 && disable >= 0 && dpms > disable {
		t.Errorf("disabled the internal panel before turning DPMS on (%d < %d)",
			disable, dpms)
	}
	journalIndex(t, "write\x1f"+paths.DisplayMode()+"\x1fexternal")

	// Only the internal panel; disabling the external one would blank the lot.
	for _, entry := range FixtureJournal() {
		if strings.Contains(entry, "DP-3") && strings.Contains(entry, "disabled") {
			t.Errorf("external monitor was disabled too: %q", entry)
		}
	}
}

// The Lua form has to be exactly what Hyprland accepts. Verified by hand
// against 0.56.1: `disabled` is the field, while `disable` and `enabled` are
// both rejected as unknown fields, and the whole call must be one argument.
func TestDisableMonitorRendersTheAcceptedLuaForm(t *testing.T) {
	got := hyprDisableMonitor("eDP-1")
	want := `hl.monitor{ output = "eDP-1", disabled = true }`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// A monitor name is interpolated into Lua source, so a name carrying a quote
// would otherwise end the string and change the statement.
func TestDisableMonitorQuotesTheMonitorName(t *testing.T) {
	got := hyprDisableMonitor(`weird"name`)
	if strings.Contains(got, `"weird"name"`) {
		t.Errorf("monitor name was not escaped: %s", got)
	}
	if !strings.Contains(got, `\"`) {
		t.Errorf("expected an escaped quote in %s", got)
	}
}

// Nothing may reach for the old surface again. Both forms fail against 0.56,
// and one of them fails silently.
func TestNoDisplayPathUsesTheSupersededHyprctlForms(t *testing.T) {
	for _, action := range []string{"restore", "normal", "headless"} {
		restore := InstallFixture(map[string]string{"env.XDG_RUNTIME_DIR": "/run"})
		if _, err := SetDisplayMode(action); err != nil {
			t.Errorf("%s: %v", action, err)
		}
		for _, entry := range FixtureJournal() {
			if strings.Contains(entry, "hyprctl\x1fdispatch\x1fdpms\x1f") {
				t.Errorf("%s still uses the pre-0.56 dispatch form: %q", action, entry)
			}
			if strings.Contains(entry, "hyprctl\x1fkeyword\x1f") {
				t.Errorf("%s still uses keyword, which fails with exit 0: %q", action, entry)
			}
		}
		restore()
	}
}
