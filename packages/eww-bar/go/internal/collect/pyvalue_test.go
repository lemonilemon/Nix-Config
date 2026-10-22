package collect

import (
	"encoding/json"
	"testing"
)

// The differential gate cannot reach these branches. diffgen decodes every
// argument with UseNumber, matching how the daemon will, so a JSON number never
// arrives as float64 there -- yet a future caller inside the daemon that uses a
// plain json.Unmarshal would produce exactly that. The float64 gap was a real
// bug once: without the case, pyTruthy fell through to `return true` and a JSON
// 0 read as truthy, which made SplitMonitors treat `"disabled": 0` as disabled.
// Table tests are the only thing that can hold these.

func TestPyTruthyAcrossNumericRepresentations(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  bool
	}{
		{"nil", nil, false},
		{"empty string", "", false},
		{"string", "x", true},
		{"string zero is not falsy", "0", true},
		{"true", true, true},
		{"false", false, false},
		{"json.Number 0", json.Number("0"), false},
		{"json.Number 0.0", json.Number("0.0"), false},
		{"json.Number 1", json.Number("1"), true},
		{"json.Number -1", json.Number("-1"), true},
		// The representations a plain json.Unmarshal produces.
		{"float64 0", float64(0), false},
		{"float64 1", float64(1), true},
		{"float64 -0", float64(-0), false},
		{"int 0", int(0), false},
		{"int 3", int(3), true},
		{"int64 0", int64(0), false},
		{"int64 3", int64(3), true},
		{"empty slice", []any{}, false},
		{"slice", []any{1}, true},
		{"empty map", map[string]any{}, false},
		{"map", map[string]any{"a": 1}, true},
	}
	for _, tc := range cases {
		if got := pyTruthy(tc.value); got != tc.want {
			t.Errorf("pyTruthy(%s = %#v) = %v, want %v", tc.name, tc.value, got, tc.want)
		}
	}
}

func TestPyEqualsIntAcrossNumericRepresentations(t *testing.T) {
	cases := []struct {
		value any
		want  int64
		equal bool
	}{
		{json.Number("3"), 3, true},
		{json.Number("3.0"), 3, true}, // Python: 3.0 == 3
		{json.Number("3.5"), 3, false},
		{json.Number("4"), 3, false},
		{"3", 3, false},   // Python: "3" == 3 is False
		{"", 3, false},    //
		{true, 1, true},   // Python: True == 1
		{true, 3, false},  //
		{false, 0, true},  // Python: False == 0
		{false, 1, false}, //
		{nil, 0, false},   // Python: None == 0 is False
		{[]any{3}, 3, false},
		{map[string]any{}, 0, false},
	}
	for _, tc := range cases {
		if got := pyEqualsInt(tc.value, tc.want); got != tc.equal {
			t.Errorf("pyEqualsInt(%#v, %d) = %v, want %v", tc.value, tc.want, got, tc.equal)
		}
	}
}

func TestPyStrDistinguishesIntFromFloat(t *testing.T) {
	// The reason ParseHistoryItems decodes with UseNumber: json.loads gives 5
	// for "5" and 5.0 for "5.0", and str() of those differ. Decoding into
	// float64 would collapse both to "5".
	cases := []struct {
		value any
		want  string
	}{
		{json.Number("5"), "5"},
		{json.Number("5.0"), "5.0"},
		{nil, "None"},
		{true, "True"},
		{false, "False"},
		{"already a string", "already a string"},
	}
	for _, tc := range cases {
		if got := pyStr(tc.value); got != tc.want {
			t.Errorf("pyStr(%#v) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
