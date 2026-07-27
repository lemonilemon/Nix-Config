package pyjson

import "testing"

// The differential gate feeds this package Ordered values decoded from JSON, so
// it checks the encoder but never the STRUCT path -- and a struct is what the
// daemon's state will be. Field order, json tags, unexported and skipped
// fields, and nil slices are all struct-only behaviour, so they live here.
//
// The expected strings are what
// json.dumps(..., separators=(",", ":"), ensure_ascii=False) produces for the
// equivalent Python dict.

type inner struct {
	Name   string   `json:"name"`
	Values []int    `json:"values"`
	Tags   []string `json:"tags"`
}

type outer struct {
	Text     string `json:"text"`
	Count    int    `json:"count"`
	Nested   inner  `json:"nested"`
	Items    []inner
	Skipped  string `json:"-"`
	unexpo   string //nolint:unused // deliberately unexported, must not appear
	Untagged bool
}

func TestEncodeStructPreservesDeclarationOrder(t *testing.T) {
	value := outer{
		Text:  "hi",
		Count: 2,
		Nested: inner{
			Name:   "n",
			Values: nil,        // must render [] not null
			Tags:   []string{}, // already empty
		},
		Items:    nil, // must render [] not null
		Skipped:  "should not appear",
		unexpo:   "should not appear",
		Untagged: true,
	}
	// Declaration order, not alphabetical: text, count, nested, Items,
	// Untagged. A field with no json tag keeps its Go name.
	want := `{"text":"hi","count":2,"nested":{"name":"n","values":[],"tags":[]},"Items":[],"Untagged":true}`

	got, err := Encode(value, false)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got != want {
		t.Errorf("Encode()\n got %s\nwant %s", got, want)
	}
}

func TestEncodeStructEscapesPerEnsureASCII(t *testing.T) {
	value := struct {
		Glyph string `json:"glyph"`
	}{Glyph: "\U000F0084"} // the battery default glyph

	raw, err := Encode(value, false)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if raw != "{\"glyph\":\"\U000F0084\"}" {
		t.Errorf("ensure_ascii=False gave %q, want the raw glyph", raw)
	}

	escaped, err := Encode(value, true)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if escaped != `{"glyph":"\udb80\udc84"}` {
		t.Errorf("ensure_ascii=True gave %s, want a surrogate pair", escaped)
	}
}

func TestEncodeRefusesNonFiniteFloats(t *testing.T) {
	// CPython emits NaN/Infinity, which is not valid JSON and which eww would
	// reject. Failing loudly beats emitting a snapshot that breaks the bar.
	for _, value := range []float64{
		positiveInf(), negativeInf(), notANumber(),
	} {
		if _, err := Encode(value, false); err == nil {
			t.Errorf("Encode(%v) should have failed", value)
		}
	}
}

func positiveInf() float64 { return 1 / zero() }
func negativeInf() float64 { return -1 / zero() }
func notANumber() float64  { return zero() / zero() }
func zero() float64        { return 0 }
