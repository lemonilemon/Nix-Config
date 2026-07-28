package collect

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

// PyFloat is Python's float(str): the conversion number_value falls back to
// when a report carries a number as a string.
//
// strconv.ParseFloat is not it. Four divergences, each verified against CPython
// 3.14 rather than assumed:
//
//   - Whitespace. float(" 1.5 ") is 1.5; ParseFloat rejects it. The set float()
//     strips is str.isspace() MINUS U+001C-U+001F, so it is neither Go's
//     unicode.IsSpace nor this package's Strip. See isPyFloatSpace.
//   - Underscores. float("1_000") is 1000.0, permitted between digits only.
//     Go allows them only alongside a base prefix, so ParseFloat("1_000") errors.
//   - Hex floats. ParseFloat("0x1p-2") is 0.25; float() raises. This is the
//     dangerous direction -- Go silently reading a value where the original
//     returned 0 -- so it is rejected explicitly rather than left to chance.
//   - Non-ASCII decimal digits. float("١٢٣") is 123.0: CPython maps any
//     Unicode Nd rune through its decimal value. ParseFloat sees garbage.
//
// Reports ok=false exactly where float() raises ValueError.
func PyFloat(text string) (float64, bool) {
	text = trimPyFloatSpace(text)
	if text == "" {
		return 0, false
	}
	text = asciiFoldDigits(text)

	// Reject the hex-literal forms Go accepts and CPython does not. Checked
	// after the sign, which both allow.
	body := text
	if body[0] == '+' || body[0] == '-' {
		body = body[1:]
	}
	if len(body) >= 2 && body[0] == '0' && (body[1] == 'x' || body[1] == 'X') {
		return 0, false
	}

	// CPython takes a signed NaN and discards the sign; ParseFloat accepts
	// "nan" but rejects "+nan" and "-nan". Caught by the gate, not predicted.
	if len(text) > 1 && (text[0] == '+' || text[0] == '-') && strings.EqualFold(body, "nan") {
		return math.NaN(), true
	}

	text, ok := stripPyUnderscores(text)
	if !ok {
		return 0, false
	}

	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		// ErrRange is not a rejection: CPython's float("1e999") is inf, and
		// ParseFloat returns ±Inf alongside the error in exactly that case.
		if numErr, isNum := err.(*strconv.NumError); isNum && numErr.Err == strconv.ErrRange {
			return value, true
		}
		return 0, false
	}
	return value, true
}

// isPyFloatSpace is the whitespace float() skips: str.isspace() without the
// four separator controls. isPySpace includes them, because str.strip() does.
// Two Python functions, two different definitions of whitespace; keeping them
// as separate predicates is the only way that stays visible.
func isPyFloatSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return false
	}
	return isPySpace(r)
}

func trimPyFloatSpace(s string) string {
	return strings.TrimFunc(s, isPyFloatSpace)
}

// asciiFoldDigits rewrites every Unicode decimal digit as its ASCII
// counterpart, which is what CPython's float() does before parsing.
func asciiFoldDigits(s string) string {
	if isASCII(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x80 {
			b.WriteRune(r)
			continue
		}
		if digit, ok := unicodeDigitValue(r); ok {
			b.WriteByte(byte('0' + digit))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// unicodeDigitValue is unicodedata.decimal(r) for the Nd category.
//
// There is no stdlib lookup for this and no room for a 760-entry table, so it
// leans on a property of the UCD: Nd characters come in contiguous, ascending
// runs of ten, and Go's range table splits on exactly those block boundaries,
// so a range always begins at a zero digit. Verified over all 760 Nd runes in
// test_go_equivalence.py against unicodedata, not taken on faith.
func unicodeDigitValue(r rune) (int, bool) {
	for _, rg := range unicode.Nd.R16 {
		if rg.Stride == 1 && r >= rune(rg.Lo) && r <= rune(rg.Hi) {
			return int(r-rune(rg.Lo)) % 10, true
		}
	}
	for _, rg := range unicode.Nd.R32 {
		if rg.Stride == 1 && r >= rune(rg.Lo) && r <= rune(rg.Hi) {
			return int(r-rune(rg.Lo)) % 10, true
		}
	}
	return 0, false
}

// stripPyUnderscores removes the digit separators float() permits, rejecting
// the placements it does not: an underscore must sit between two digits, so
// "1_000" is 1000.0 while "_1", "1_", and "1__0" all raise.
func stripPyUnderscores(s string) (string, bool) {
	if !strings.ContainsRune(s, '_') {
		return s, true
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '_' {
			b.WriteByte(s[i])
			continue
		}
		if i == 0 || i == len(s)-1 || !isASCIIDigit(s[i-1]) || !isASCIIDigit(s[i+1]) {
			return "", false
		}
	}
	return b.String(), true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }
