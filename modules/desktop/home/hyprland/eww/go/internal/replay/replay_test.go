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
	for i, r := range records {
		byFunction[r.Fn]++

		if recordsSupersededHyprctl(r) {
			superseded++
			continue
		}

		got, err := Answer(Call{Fn: r.Fn, Args: r.Args})
		if err != nil {
			failures = append(failures,
				fmt.Sprintf("%s%s: errored: %v", r.Fn, argsOf(r), err))
			continue
		}
		gotJSON, err := canonical(got)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: unencodable: %v", r.Fn, err))
			continue
		}
		var want any
		if err := json.Unmarshal(r.Value, &want); err != nil {
			t.Fatalf("record %d has an unreadable value: %v", i, err)
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
