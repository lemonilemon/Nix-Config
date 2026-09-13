package collect

import (
	"strings"
	"unicode"
)

// PyLower is Python's str.lower(). strings.ToLower agrees on every code point but
// U+0130, the capital I with dot above, which lowercases to TWO characters in
// Python -- "i" followed by U+0307 -- where Go yields a bare "i". The AI
// collectors lowercase agent keys that are then rendered, so a dropped combining
// mark would reach the popup.

// Escapes, not literals: dottedLowerI is "i" plus a COMBINING DOT ABOVE, which
// renders as one glyph.
const (
	capitalIWithDot     = "\u0130"
	capitalIWithDotRune = '\u0130'
	dottedLowerI        = "i\u0307"
)

func PyLower(s string) string {
	if strings.Contains(s, capitalIWithDot) {
		s = strings.ReplaceAll(s, capitalIWithDot, dottedLowerI)
	}
	return strings.ToLower(s)
}

// PyTitle is Python's str.title(). The rule is a running flag: each character is
// titlecased if the PREVIOUS character was not cased, and lowercased otherwise, so
// any digit or punctuation restarts a word -- "max_5x" becomes "Max 5X" and
// "don't" becomes "Don'T". Both are real outputs of the plan formatter.
//
// Both branches use Unicode's FULL case mappings, which are one-to-many and so
// cannot come from unicode.ToTitle/ToLower alone.
func PyTitle(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	previousIsCased := false
	for _, r := range s {
		switch {
		case previousIsCased && r == capitalIWithDotRune:
			out.WriteString(dottedLowerI)
		case previousIsCased:
			out.WriteRune(unicode.ToLower(r))
		default:
			if full, isFull := fullTitleCase[r]; isFull {
				out.WriteString(full)
			} else {
				out.WriteRune(unicode.ToTitle(r))
			}
		}
		previousIsCased = isCased(r)
	}
	return out.String()
}

// isCased is Python's Py_UNICODE_ISCASED: upper, lower or title case.
func isCased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r)
}

// SplitLines is Python's str.splitlines(). strings.Split(s, "\n") is not it:
// Python also breaks on \r, \r\n, \v, \f, \x1c, \x1d, \x1e, \x85, U+2028 and
// U+2029, and emits no trailing empty element for a string ending in a terminator.
func SplitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		width := lineBreakWidth(runes, i)
		if width == 0 {
			continue
		}
		lines = append(lines, string(runes[start:i]))
		i += width - 1
		start = i + 1
	}
	if start < len(runes) {
		lines = append(lines, string(runes[start:]))
	}
	return lines
}

// lineBreakWidth reports how many runes of line terminator start at i, or 0.
func lineBreakWidth(runes []rune, i int) int {
	switch runes[i] {
	case '\r':
		if i+1 < len(runes) && runes[i+1] == '\n' {
			return 2
		}
		return 1
	case '\n', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
		return 1
	}
	return 0
}

// SplitWhitespaceN is Python's str.split(maxsplit=n) with no separator: splits on
// runs of whitespace, ignores leading and trailing, and -- the part strings.Fields
// cannot express -- stops after n splits and hands back the rest of the string as
// the final element, internal spacing intact. n < 0 means unlimited.
func SplitWhitespaceN(s string, n int) []string {
	var fields []string
	i := 0
	runes := []rune(s)
	for {
		for i < len(runes) && isPySpace(runes[i]) {
			i++
		}
		if i >= len(runes) {
			break
		}
		if n >= 0 && len(fields) == n {
			// The remainder is one field, taken verbatim. Its leading whitespace was
			// consumed above, but its trailing whitespace is KEPT:
			// '  one  '.split(maxsplit=0) is ['one  '], not ['one'].
			fields = append(fields, string(runes[i:]))
			break
		}
		start := i
		for i < len(runes) && !isPySpace(runes[i]) {
			i++
		}
		fields = append(fields, string(runes[start:i]))
	}
	return fields
}

// Strip is Python's str.strip(). strings.TrimSpace is close but not equal: Go's
// unicode.IsSpace excludes U+001C-U+001F, which Python's str.isspace() includes.
func Strip(s string) string {
	runes := []rune(s)
	start, end := 0, len(runes)
	for start < end && isPySpace(runes[start]) {
		start++
	}
	for end > start && isPySpace(runes[end-1]) {
		end--
	}
	return string(runes[start:end])
}

// isPySpace matches Python's str.isspace(), which both str.split() and str.strip()
// use. Code points rather than literals because several are invisible.
func isPySpace(r rune) bool {
	switch {
	case r >= 0x09 && r <= 0x0d: // tab, LF, VT, FF, CR
		return true
	case r >= 0x1c && r <= 0x1f: // file/group/record/unit separators
		return true
	case r == ' ' || r == 0x85 || r == 0xa0: // space, NEL, NBSP
		return true
	case r == 0x1680: // ogham space mark
		return true
	case r >= 0x2000 && r <= 0x200a: // en quad .. hair space
		return true
	case r == 0x2028 || r == 0x2029: // line/paragraph separator
		return true
	case r == 0x202f || r == 0x205f || r == 0x3000: // narrow/medium/ideographic
		return true
	}
	return false
}
