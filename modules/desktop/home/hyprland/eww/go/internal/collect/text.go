// Package collect holds the bar's collectors.
package collect

import "unicode/utf8"

// TruncateText counts CODE POINTS, not bytes: thirteen call sites pass Nerd Font
// glyphs (U+F0xx, three bytes each), which byte slicing would cut in the wrong
// place and split into invalid UTF-8.
//
// When maxLen < 3 it slices with a negative index and can return something longer
// than it was given. That is a bug, faithful to the original, and unreachable from
// any current caller.
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
