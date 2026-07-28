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
// run verified.
//
// This file is what the port's safety survives on after the Python is deleted.
// The gate could ask "does Go agree with CPython?"; once CPython is gone the
// only answerable question is "does Go still answer what it answered when we
// checked?" -- which needs these exact inputs and outputs. Regenerate with:
//
//	EWW_GOLDEN_OUT=.../golden.jsonl.gz python3 -m unittest \
//	    tests.eww_bar_backend.test_go_equivalence
//
// and only while the Python is still present to be checked against. Generate it
// under TZ=UTC: see TestMain.
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
	for i, r := range records {
		byFunction[r.Fn]++

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
