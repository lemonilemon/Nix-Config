package collect

// SplitLines is Python's str.splitlines().
//
// strings.Split(s, "\n") is not it, and the difference is not academic: Python
// also breaks on \r, \r\n, \v, \f, \x1c, \x1d, \x1e, \x85, U+2028 and U+2029,
// and it does not emit a trailing empty element for a string that ends in a
// terminator. Twenty-one call sites in the original parse subprocess output
// this way. `\r\n` alone would leave a stray \r on the end of every field --
// and the parsers that go on to .strip() would hide it, while the ones that
// index or compare would not.
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

// SplitWhitespaceN is Python's str.split(maxsplit=n) with no separator.
//
// It splits on runs of whitespace, ignores leading and trailing whitespace, and
// -- the part strings.Fields cannot express -- stops after n splits and hands
// back the rest of the string as the final element, internal spacing intact.
// bluetooth_state_from_text depends on that last part: `Device AA:BB My
// Speaker` must yield the name as one field, spaces and all.
//
// n < 0 means unlimited, matching Python's default.
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
			// The remainder is one field, taken verbatim to the end. Its
			// leading whitespace was consumed above, but its trailing
			// whitespace is KEPT: '  one  '.split(maxsplit=0) is ['one  '],
			// not ['one']. Trimming it here looked obviously right and was
			// wrong -- the equivalence gate caught it on a bluetooth device
			// name with a trailing space.
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

// Strip is Python's str.strip() with no argument.
//
// strings.TrimSpace is close but not equal: Go's unicode.IsSpace excludes
// U+001C-U+001F, which Python's str.isspace() includes. Spelled out here so the
// two definitions of "whitespace" in this package cannot drift apart.
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

// isPySpace matches Python's str.isspace(), which is what both str.split() and
// str.strip() use. Written as code points because several of these are
// invisible and a literal would be unreviewable.
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
