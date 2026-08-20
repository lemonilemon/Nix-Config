package replay

import (
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"ewwbar/internal/collect"
)

// golden holds every case the differential gate generated, with the answer that
// run verified against CPython: 42,621 cases over 97 entry points.
//
// This file is what the port's safety survives on. The gate could ask "does Go
// agree with CPython?"; with CPython gone the only answerable question is "does
// Go still answer what it answered when we checked?" -- which needs these exact
// inputs and outputs.
//
// It CANNOT BE REGENERATED. The generator was
// tests/eww_bar_backend/test_go_equivalence.py, driven as
//
//	EWW_GOLDEN_OUT=.../golden.jsonl.gz python3 -m unittest \
//	    tests.eww_bar_backend.test_go_equivalence
//
// and it was deleted with the rest of the Python backend. Recovering it means
// going back through history:
//
//	git log --diff-filter=D -- tests/eww_bar_backend/test_go_equivalence.py
//	git show <that commit>^:tests/eww_bar_backend/test_go_equivalence.py
//
// So treat a failure here as a regression in the Go, never as a stale recording
// to be refreshed. Adding coverage means writing an ordinary Go test, not
// appending to this file.
//
// Recorded under TZ=UTC, which TestMain re-pins.
//
//go:embed testdata/golden.jsonl.gz
var golden []byte

func TestMain(m *testing.M) {
	// UTC, pinned, and this is load-bearing. Period labels, clock fields and
	// every reset time resolve through the local zone, so the recorded answers
	// are only meaningful in the zone they were recorded in. The file is
	// generated under TZ=UTC; without this the suite passes on the machine that
	// generated it and fails everywhere else -- which is exactly how it first
	// showed up, green locally and red in the Nix sandbox where there is no
	// /etc/localtime.
	time.Local = time.UTC

	// The same rail package control and package watch use: several recorded
	// calls are fixture-driven collectors, and a collector shells out.
	restore := collect.InstallFixture(map[string]string{})
	code := m.Run()
	restore()
	os.Exit(code)
}

// canonical renders a value so two JSON encodings of the same data compare
// equal regardless of how they were produced.
//
// Needed because the two sides encode differently: a Go answer may be a struct,
// which marshals in field order, while the same value read back from the golden
// file is a map, which marshals sorted. Round-tripping both through `any`
// puts them in the same form.
func canonical(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return "", err
	}
	recoded, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	return string(recoded), nil
}

type record struct {
	Fn    string            `json:"fn"`
	Args  []json.RawMessage `json:"args"`
	Value json.RawMessage   `json:"value"`
}

func loadGolden(t *testing.T) []record {
	t.Helper()
	reader, err := gzip.NewReader(strings.NewReader(string(golden)))
	if err != nil {
		t.Fatalf("golden file unreadable: %v", err)
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	var records []record
	for {
		var r record
		if err := decoder.Decode(&r); err != nil {
			break
		}
		records = append(records, r)
	}
	return records
}

func TestGoldenReplay(t *testing.T) {
	records := loadGolden(t)
	if len(records) < 40000 {
		t.Fatalf("golden file holds only %d cases; it was regenerated from a "+
			"partial run", len(records))
	}

	byFunction := map[string]int{}
	var failures []string
	superseded := 0
	supersededNetwork := 0
	for i, r := range records {
		byFunction[r.Fn]++

		if recordsSupersededHyprctl(r) {
			superseded++
			continue
		}
		if recordsSupersededNetwork(r) {
			supersededNetwork++
			continue
		}

		got, err := Answer(Call{Fn: r.Fn, Args: r.Args})
		if err != nil {
			failures = append(failures,
				fmt.Sprintf("%s%s: errored: %v", r.Fn, argsOf(r), err))
			continue
		}
		var want any
		if err := json.Unmarshal(r.Value, &want); err != nil {
			t.Fatalf("record %d has an unreadable value: %v", i, err)
		}

		// The one place the recording is allowed to be incomplete rather than
		// wrong. See narrowSnapshot.
		if r.Fn == "ControlHandle" {
			want, got = narrowSnapshot(want, got)
		}

		gotJSON, err := canonical(got)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: unencodable: %v", r.Fn, err))
			continue
		}
		wantJSON, err := canonical(want)
		if err != nil {
			t.Fatalf("record %d value unencodable: %v", i, err)
		}
		if gotJSON != wantJSON {
			failures = append(failures, fmt.Sprintf(
				"%s%s\n    want %s\n    got  %s",
				r.Fn, argsOf(r), truncate(wantJSON), truncate(gotJSON)))
		}
	}

	if len(failures) > 0 {
		shown := failures
		if len(shown) > 10 {
			shown = shown[:10]
		}
		t.Fatalf("%d/%d recorded cases disagree:\n%s",
			len(failures), len(records), strings.Join(shown, "\n"))
	}

	// A guard against the file silently narrowing: the port covers 97 distinct
	// entry points, and a regeneration that lost most of them would still pass
	// every case it kept.
	if len(byFunction) < 90 {
		t.Fatalf("golden file covers only %d functions", len(byFunction))
	}

	// And a guard on the skip itself: if the display code is ever migrated
	// again, or these cases are somehow lost, the count moving is the signal.
	if superseded != supersededHyprctlCases {
		t.Errorf("skipped %d superseded-hyprctl cases, expected %d -- if the "+
			"display code changed again, update the constant and the Go tests "+
			"that replaced these cases", superseded, supersededHyprctlCases)
	}

	// And a guard on the narrowing, for the same reason: narrowSnapshot is the
	// only concession the golden file makes to state that grew after the
	// recording, and it applies to exactly these cases.
	if supersededNetwork != supersededNetworkCases {
		t.Errorf("skipped %d superseded-network cases, expected %d -- if the "+
			"network control command changed again, update the constant and "+
			"the Go tests that replaced these cases",
			supersededNetwork, supersededNetworkCases)
	}

	if byFunction["ControlHandle"] != controlHandleCases {
		t.Errorf("golden holds %d ControlHandle cases, expected %d -- "+
			"narrowSnapshot applies to these and nothing else",
			byFunction["ControlHandle"], controlHandleCases)
	}
}

// controlHandleCases is how many recorded cases bundle a whole bar snapshot
// into their answer. Pinned for the same reason supersededHyprctlCases is: the
// narrowing below applies to exactly these, and it must not spread.
const controlHandleCases = 46

// narrowSnapshot restricts a ControlHandle answer to the state keys the
// recording actually has an opinion about.
//
// ControlHandle is the only recorded function that carries a whole bar snapshot
// in its answer, and that snapshot is a photograph of the state as it stood
// while CPython was still around. Any field added to the bar afterwards changes
// all 46 recorded answers at once, in a way no amount of correctness in the Go
// can avoid -- the network popup's connectivity, connection identity and speed
// card were the first, and they will not be the last.
//
// So the recording keeps every assertion it can still make and loses only the
// one it cannot: that no field was ever added. A key it recorded is compared
// exactly as before, so a changed value, a deleted key and a rename all still
// fail. A key it never saw is dropped from the comparison.
//
// Both sides are re-encoded through the same marshaller afterwards. The
// recorded snapshot is a string holding CPython's own byte-for-byte encoding,
// which pyjson reproduces; re-encoding only one side would compare Go's
// separators against Python's and fail every case for a reason that has nothing
// to do with the state.
//
// This is NOT the sanctioned skip above and must not become it. Those 13 cases
// are dropped whole because the recording is wrong about the world; these 46
// stay green on everything they recorded, including their reply and their
// side-effect journal. New state fields get ordinary Go tests, exactly as the
// golden file's own note requires.
func narrowSnapshot(recorded, produced any) (any, any) {
	recordedMap, recordedOK := recorded.(map[string]any)
	producedMap, producedOK := produced.(map[string]any)
	if !recordedOK || !producedOK {
		return recorded, produced
	}
	recordedText, recordedOK := recordedMap["snapshot"].(string)
	producedText, producedOK := producedMap["snapshot"].(string)
	if !recordedOK || !producedOK {
		return recorded, produced
	}

	var recordedState, producedState any
	if json.Unmarshal([]byte(recordedText), &recordedState) != nil ||
		json.Unmarshal([]byte(producedText), &producedState) != nil {
		return recorded, produced
	}

	recordedEncoded, recordedErr := json.Marshal(recordedState)
	producedEncoded, producedErr := json.Marshal(projectOnto(recordedState, producedState))
	if recordedErr != nil || producedErr != nil {
		return recorded, produced
	}

	return withSnapshot(recordedMap, string(recordedEncoded)),
		withSnapshot(producedMap, string(producedEncoded))
}

// withSnapshot copies an answer with its snapshot field replaced, leaving the
// reply and journal beside it untouched.
func withSnapshot(answer map[string]any, snapshot string) map[string]any {
	out := make(map[string]any, len(answer))
	for key, value := range answer {
		out[key] = value
	}
	out["snapshot"] = snapshot
	return out
}

// projectOnto returns produced with every key the recorded value does not
// mention removed, recursively.
//
// A key the recording HAS is never synthesised when produced lacks it, which is
// what keeps a deleted or renamed field failing: the recorded side still holds
// it and the projected side does not.
func projectOnto(recorded, produced any) any {
	if recordedMap, ok := recorded.(map[string]any); ok {
		if producedMap, ok := produced.(map[string]any); ok {
			out := make(map[string]any, len(recordedMap))
			for key, recordedValue := range recordedMap {
				producedValue, present := producedMap[key]
				if !present {
					continue
				}
				out[key] = projectOnto(recordedValue, producedValue)
			}
			return out
		}
		return produced
	}

	if recordedList, ok := recorded.([]any); ok {
		producedList, ok := produced.([]any)
		// A length change is a real disagreement, so it is left alone to fail.
		if ok && len(recordedList) == len(producedList) {
			out := make([]any, len(producedList))
			for i := range producedList {
				out[i] = projectOnto(recordedList[i], producedList[i])
			}
			return out
		}
	}

	return produced
}

// supersededHyprctlCases is how many recorded cases encode hyprctl's pre-0.56
// command line. Pinned so the skip cannot quietly widen.
const supersededHyprctlCases = 13

// recordsSupersededHyprctl reports whether a recorded case asserts a hyprctl
// invocation that Hyprland no longer accepts.
//
// THIS IS THE ONLY SANCTIONED SKIP, and it exists because the recording can be
// wrong about the world in a way that re-running it cannot fix. These 13 cases
// pin the argv of `hyprctl dispatch dpms on|off` and `hyprctl keyword monitor
// <name>,disable`. Hyprland 0.56 moved that surface to Lua: the first is now a
// syntax error inside the generated hl.dispatch(...) and exits 7, and the
// second answers "keyword can't work with non-legacy parsers" while exiting 0.
// Keeping these cases green would mean keeping internal/collect/display.go
// calling commands that cannot work -- which is exactly the bug that made every
// display-mode button silently do nothing.
//
// The recording is NOT edited. It stays the byte-exact artifact the port was
// verified against; these cases are stepped over here, where the reason is
// visible, and their coverage is replaced by ordinary Go tests in
// internal/collect (TestDisplayModeUsesTheLuaHyprctlSurface and friends), which
// is what the file header asks for anyway.
//
// Everything else in the file still means what it always meant: a failure is a
// regression in the Go, never a stale recording to be refreshed.
func recordsSupersededHyprctl(r record) bool {
	if r.Fn != "SetDisplayMode" && r.Fn != "ControlHandle" {
		return false
	}
	// Matched against the RAW JSON of the recorded value, where the journal's
	// 0x1f separators are written as the six literal characters \u001f. Raw
	// string literals below so those stay six characters and are not unescaped
	// by the Go compiler into the byte itself.
	for _, old := range [...]string{
		`hyprctl\u001fdispatch\u001fdpms\u001fon`,
		`hyprctl\u001fdispatch\u001fdpms\u001foff`,
		`hyprctl\u001fkeyword\u001fmonitor\u001f`,
	} {
		if strings.Contains(string(r.Value), old) {
			return true
		}
	}
	return false
}

// supersededNetworkCases is how many recorded cases pin the `network` control
// command's behaviour from before it gained a second action. Pinned so this
// skip cannot quietly widen, exactly as supersededHyprctlCases is.
const supersededNetworkCases = 2

// recordsSupersededNetwork reports whether a recorded case asserts something
// about the `network` control command that the speed test legitimately changed.
//
// Two cases, and neither is a snapshot difference -- narrowSnapshot already
// absorbs those. What changed here is behaviour the recording is right to
// notice:
//
//   - `network wifi-toggle` records the side-effect journal, and the journal
//     grew. ToggleWifi re-reads the network state on the way out, and that read
//     now also asks nmcli which connection is active and what NetworkManager
//     makes of its connectivity. The extra forks are the feature.
//   - `network bogus` records the usage error, which said "network action must
//     be wifi-toggle". It cannot keep saying that while speedtest is also
//     valid; a usage message that omits half the actions is worse than one the
//     recording does not recognise.
//
// Their coverage is replaced by ordinary Go tests in internal/control
// (TestNetworkSpeedtestQueues and friends), which is what the file header asks
// for. Everything else in the file still means what it always meant.
func recordsSupersededNetwork(r record) bool {
	if r.Fn != "ControlHandle" || len(r.Args) < 2 {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal(r.Args[1], &payload); err != nil {
		return false
	}
	return payload["command"] == "network"
}

func argsOf(r record) string {
	parts := make([]string, 0, len(r.Args))
	for _, a := range r.Args {
		parts = append(parts, truncate(string(a)))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func truncate(s string) string {
	if len(s) <= 120 {
		return s
	}
	return s[:117] + "..."
}
