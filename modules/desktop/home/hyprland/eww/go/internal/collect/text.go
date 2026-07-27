// Package collect holds the ported bar collectors.
//
// Every function here is a transliteration of one in
// scripts/eww_bar_backend/, and must agree with it byte for byte -- the two run
// side by side until the daemon flips over, and diffgen_test.go checks them
// against each other on generated input.
package collect

import "unicode/utf8"

// TruncateText mirrors common.truncate_text.
//
// Two things about the original are load-bearing and neither is visible in its
// three lines:
//
// Python's len() and slicing count code points, Go's count bytes. Thirteen call
// sites pass Nerd Font glyphs (U+F0xx, three bytes each), so byte slicing would
// both cut at the wrong place and split a glyph into invalid UTF-8 -- and
// because the limits are generous, it would fire on a long ASCII device name
// while every test with a short one passed.
//
// And when maxLen < 3 the original slices with a negative index: text[:maxLen-3]
// becomes text[:len(text)+maxLen-3], so it keeps all but the last few code
// points and appends "...", returning something *longer* than it was given.
// That looks like a bug, and it is one, but it is reachable only from a caller
// passing a tiny limit and no caller does; reproducing it costs three lines and
// removes any doubt about whether a difference here is intentional.
func TruncateText(text string, maxLen int) string {
	if utf8.RuneCountInString(text) <= maxLen {
		return text
	}
	runes := []rune(text)
	cut := maxLen - 3
	if cut < 0 {
		// Python's s[:k] for negative k is s[:max(0, len(s)+k)].
		cut = len(runes) + cut
		if cut < 0 {
			cut = 0
		}
	}
	return string(runes[:cut]) + "..."
}
