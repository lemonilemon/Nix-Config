package collect

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// ParseISOEpoch runs two parsers in sequence: fromisoformat, then a strict
// strptime over the first 19 characters. They do not accept the same language --
// fromisoformat demands zero-padded fields where strptime's %m/%d/%H take one or
// two digits -- so dropping either changes which timestamps resolve.
func ParseISOEpoch(value any) (float64, bool) {
	text, isString := value.(string)
	if !isString || text == "" {
		return 0, false
	}

	if micros, ok := parseISOFormat(strings.ReplaceAll(text, "Z", "+00:00")); ok {
		return microsToSeconds(micros), true
	}

	// The fallback re-reads the ORIGINAL text, not the Z-substituted one, and
	// slices it by code point -- Python's value[:19] counts characters.
	runes := []rune(text)
	if len(runes) > 19 {
		runes = runes[:19]
	}
	if seconds, ok := parseStrptimeSeconds(string(runes)); ok {
		return float64(seconds), true
	}
	return 0, false
}

// FormatClockTime takes any rather than float64 because the None branch is part of
// the contract. A non-numeric, non-nil argument returns "--" where CPython raises.
func FormatClockTime(epoch any) string {
	if epoch == nil {
		return "--"
	}
	seconds, ok := pyNumber(epoch)
	if !ok {
		return "--"
	}
	// time.localtime floors: localtime(-0.5) is one second before the epoch.
	return time.Unix(int64(math.Floor(seconds)), 0).Local().Format("2006-01-02 15:04")
}

// microsToSeconds is datetime.timestamp(). Whole seconds and the sub-second
// remainder are converted separately: converting the total rounds twice once the
// count passes 2^53, which the gate caught at 9998-12-31.
func microsToSeconds(micros int64) float64 {
	// Go truncates both operators toward zero, so the two parts keep a
	// consistent sign and add back correctly for negative epochs.
	return float64(micros/1e6) + float64(micros%1e6)/1e6
}

// parseISOFormat is datetime.fromisoformat, returning microseconds since the Unix
// epoch. Microseconds rather than a time.Time so fractional UTC offsets survive:
// fromisoformat accepts "+23:59:59.999999", which time.FixedZone cannot hold.
//
// The accepted grammar was characterised against CPython 3.14, not read off ISO
// 8601. Wider than RFC 3339: the date/time separator is any single character, week
// dates parse, "T24:00:00" rolls to the next midnight, basic and extended forms
// are both allowed. Narrower: "2024-01", bare years, ordinal dates and any leading
// or trailing space all raise.
//
// KNOWN LIMIT within one UTC offset of datetime.min or datetime.max, where CPython
// probes the platform's localtime and Go computes the arithmetic answer instead.
func parseISOFormat(s string) (int64, bool) {
	year, month, day, rest, ok := parseISODate(s)
	if !ok {
		return 0, false
	}
	if rest == "" {
		if day == 0 {
			return 0, false
		}
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local).Unix() * 1e6, true
	}

	// Exactly one separator character, and it may be anything at all --
	// "2024-01-15X10:30:00" parses. Consumed as a rune, not a byte.
	_, width := utf8.DecodeRuneInString(rest)
	clock := rest[width:]
	if clock == "" {
		return 0, false
	}

	hour, minute, second, micros, offset, hasOffset, ok := parseISOClock(clock)
	if !ok {
		return 0, false
	}
	// Hour 24 is midnight of the following day, and only when nothing else is
	// set; time.Date normalises the rollover.
	if hour == 24 && (minute != 0 || second != 0 || micros != 0) {
		return 0, false
	}
	// Hour 24 also relaxes the day's LOWER bound to zero: "2024-01-00T24:00:00" is
	// 1 January while "2024-01-00T00:00:00" raises. The upper bound does not move.
	if day == 0 && hour != 24 {
		return 0, false
	}

	if hasOffset {
		utc := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
		return utc.Unix()*1e6 + micros - offset, true
	}
	local := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
	return local.Unix()*1e6 + micros, true
}

// parseISODate consumes the date at the head of s and returns the remainder.
//
// Accepted: YYYY-MM-DD, YYYYMMDD, YYYY-Www[-D], YYYYWww[D]. Rejected, and each
// verified: YYYY-MM, a bare year, and ordinal dates like 2024-366.
func parseISODate(s string) (int, int, int, string, bool) {
	if len(s) < 7 {
		return 0, 0, 0, "", false
	}
	year, ok := atoiFixed(s[:4])
	if !ok {
		return 0, 0, 0, "", false
	}

	extended := s[4] == '-'
	head := 4
	if extended {
		head = 5
	}
	if head >= len(s) {
		return 0, 0, 0, "", false
	}

	if s[head] == 'W' {
		return parseISOWeekDate(s, year, head, extended)
	}

	if extended {
		if len(s) < 10 || s[7] != '-' {
			return 0, 0, 0, "", false
		}
		month, okMonth := atoiFixed(s[5:7])
		day, okDay := atoiFixed(s[8:10])
		if !okMonth || !okDay || !validYMDAllowingDayZero(year, month, day) {
			return 0, 0, 0, "", false
		}
		return year, month, day, s[10:], true
	}

	if len(s) < 8 {
		return 0, 0, 0, "", false
	}
	month, okMonth := atoiFixed(s[4:6])
	day, okDay := atoiFixed(s[6:8])
	if !okMonth || !okDay || !validYMDAllowingDayZero(year, month, day) {
		return 0, 0, 0, "", false
	}
	return year, month, day, s[8:], true
}

// parseISOWeekDate handles the YYYY-Www[-D] and YYYYWww[D] forms. An absent
// weekday means Monday.
func parseISOWeekDate(s string, year, head int, extended bool) (int, int, int, string, bool) {
	if head+3 > len(s) {
		return 0, 0, 0, "", false
	}
	week, ok := atoiFixed(s[head+1 : head+3])
	if !ok {
		return 0, 0, 0, "", false
	}

	i := head + 3
	weekday := 1
	if extended {
		// "-D" is the weekday only when what FOLLOWS it is not another digit. That
		// lookahead is how CPython tells "2024-W03-3T10:00" from
		// "2024-W03-110+00:00", where the "-" is the date/time separator.
		if i+1 < len(s) && s[i] == '-' && isASCIIDigit(s[i+1]) &&
			(i+2 >= len(s) || !isASCIIDigit(s[i+2])) {
			weekday, _ = atoiFixed(s[i+1 : i+2])
			i += 2
		}
	} else if i < len(s) && isASCIIDigit(s[i]) {
		weekday, _ = atoiFixed(s[i : i+1])
		i++
	}
	if week < 1 || week > 53 || weekday < 1 || weekday > 7 {
		return 0, 0, 0, "", false
	}

	// Week 1 is the week containing 4 January. The round-trip through ISOWeek is
	// what rejects a 53rd week in a year that has only 52.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	weekMonday := jan4.AddDate(0, 0, -((int(jan4.Weekday())+6)%7)+(week-1)*7)
	if isoYear, isoWeek := weekMonday.ISOWeek(); isoYear != year || isoWeek != week {
		return 0, 0, 0, "", false
	}
	rest := s[i:]

	// DELIBERATE NARROWING, the one place this file is not fromisoformat.
	//
	// A basic week date followed immediately by another digit ("2024W03110:30") is
	// ambiguous, and CPython resolves it with a positional heuristic that does not
	// follow from the string. This rejects the shape outright rather than guessing:
	// Go declines where CPython might parse, never the reverse. The golden replay
	// pins the divergence at exactly this shape and always None.
	if !extended && rest != "" && isASCIIDigit(rest[0]) {
		return 0, 0, 0, "", false
	}

	date := weekMonday.AddDate(0, 0, weekday-1)
	return date.Year(), int(date.Month()), date.Day(), rest, true
}

// parseISOClock reads the time and optional UTC offset. micros and offset are
// both microseconds; offset is meaningful only when hasOffset.
func parseISOClock(s string) (hour, minute, second int, micros, offset int64, hasOffset, ok bool) {
	body := s
	if i := strings.IndexAny(s, "+-"); i >= 0 {
		body = s[:i]
		offset, ok = parseISOOffset(s[i:])
		if !ok {
			return 0, 0, 0, 0, 0, false, false
		}
		hasOffset = true
	} else if n := len(s); n > 0 && s[n-1] == 'Z' {
		// Unreachable from ParseISOEpoch, which substitutes Z away first. Kept
		// because fromisoformat accepts it, and this function is the grammar.
		body = s[:n-1]
		hasOffset = true
	}

	// An empty fraction is legal only when a zone follows: "10:30:00." is
	// rejected, "10:30:00.+00:00" is zero microseconds, "10:30:00.," is rejected.
	body, micros, ok = splitFraction(body, hasOffset)
	if !ok {
		return 0, 0, 0, 0, 0, false, false
	}

	// When a zone follows, a BASIC-format time body of odd length has its final
	// digit silently discarded: "T030+00:00" is 03:00 and "T0300000+00:00" is
	// 03:00:00. The same bodies are rejected with no zone, and extended forms
	// are rejected either way. Reproduced, not corrected.
	if hasOffset && !strings.Contains(body, ":") {
		if n := len(body); n == 3 || n == 5 || n == 7 {
			body = body[:n-1]
		}
	}

	hour, minute, second, ok = parseClockFields(body)
	if !ok {
		return 0, 0, 0, 0, 0, false, false
	}
	return hour, minute, second, micros, offset, hasOffset, true
}

// splitFraction peels off a "." or "," fractional-seconds suffix, truncating
// past six digits the way CPython does rather than rounding.
func splitFraction(s string, allowEmpty bool) (string, int64, bool) {
	i := strings.IndexAny(s, ".,")
	if i < 0 {
		return s, 0, true
	}
	digits := s[i+1:]
	if digits == "" {
		if !allowEmpty {
			return "", 0, false
		}
		return s[:i], 0, true
	}
	for j := 0; j < len(digits); j++ {
		if !isASCIIDigit(digits[j]) {
			return "", 0, false
		}
	}
	if len(digits) > 6 {
		digits = digits[:6]
	}
	for len(digits) < 6 {
		digits += "0"
	}
	micros, ok := atoiFixed(digits)
	if !ok {
		return "", 0, false
	}
	return s[:i], int64(micros), true
}

// parseClockFields reads HH, HH:MM, HHMM, HH:MM:SS or HHMMSS. Mixing the basic
// and extended forms ("10:3000") is rejected, as CPython rejects it.
func parseClockFields(s string) (int, int, int, bool) {
	var fields []string
	switch len(s) {
	case 2:
		fields = []string{s}
	case 4:
		fields = []string{s[0:2], s[2:4]}
	case 5:
		if s[2] != ':' {
			return 0, 0, 0, false
		}
		fields = []string{s[0:2], s[3:5]}
	case 6:
		fields = []string{s[0:2], s[2:4], s[4:6]}
	case 8:
		if s[2] != ':' || s[5] != ':' {
			return 0, 0, 0, false
		}
		fields = []string{s[0:2], s[3:5], s[6:8]}
	default:
		return 0, 0, 0, false
	}

	var parts [3]int
	for i, field := range fields {
		value, ok := atoiFixed(field)
		if !ok {
			return 0, 0, 0, false
		}
		parts[i] = value
	}
	hour, minute, second := parts[0], parts[1], parts[2]
	if hour > 24 || minute > 59 || second > 59 {
		return 0, 0, 0, false
	}
	return hour, minute, second, true
}

// parseISOOffset reads ±HH, ±HHMM, ±HH:MM, ±HHMMSS, ±HH:MM:SS or
// ±HH:MM:SS.ffffff into microseconds. CPython requires |offset| < 24h.
func parseISOOffset(s string) (int64, bool) {
	if len(s) < 3 {
		return 0, false
	}
	sign := int64(1)
	if s[0] == '-' {
		sign = -1
	}
	body, micros, ok := splitFraction(s[1:], false)
	if !ok {
		return 0, false
	}
	hour, minute, second, ok := parseClockFields(body)
	if !ok {
		return 0, false
	}
	total := int64(hour)*3600e6 + int64(minute)*60e6 + int64(second)*1e6 + micros
	if total >= 24*3600e6 {
		return 0, false
	}
	return sign * total, true
}

// parseStrptimeSeconds is time.strptime(s, "%Y-%m-%dT%H:%M:%S") fed to time.mktime,
// deliberately a different grammar from parseISOFormat: %m, %d, %H, %M and %S each
// accept one OR two digits, which is what rescues "2024-1-5T1:2:3".
//
// Two behaviours are not visible in the format string: the format compiles with
// re.IGNORECASE so the literal T matches a lowercase t, and the calendar IS
// validated, so "2024-02-30" raises rather than normalising to 1 March.
func parseStrptimeSeconds(s string) (int64, bool) {
	rest := s
	year, rest, ok := takeDigits(rest, 4, 4)
	if !ok || !takeLiteral(&rest, '-') {
		return 0, false
	}
	month, rest, ok := takeDigits(rest, 1, 2)
	if !ok || month < 1 || month > 12 || !takeLiteral(&rest, '-') {
		return 0, false
	}
	day, rest, ok := takeDigits(rest, 1, 2)
	if !ok || !validYMD(year, month, day) || !takeLiteralFold(&rest, 'T') {
		return 0, false
	}
	hour, rest, ok := takeDigits(rest, 1, 2)
	if !ok || hour > 23 || !takeLiteral(&rest, ':') {
		return 0, false
	}
	minute, rest, ok := takeDigits(rest, 1, 2)
	if !ok || minute > 59 || !takeLiteral(&rest, ':') {
		return 0, false
	}
	// %S accepts 0..61: Python's directive still allows for leap seconds.
	second, rest, ok := takeDigits(rest, 1, 2)
	if !ok || second > 61 || rest != "" {
		return 0, false
	}
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local).Unix(), true
}

// takeDigits consumes between min and max ASCII digits, greedily.
func takeDigits(s string, minDigits, maxDigits int) (int, string, bool) {
	n := 0
	for n < maxDigits && n < len(s) && isASCIIDigit(s[n]) {
		n++
	}
	if n < minDigits {
		return 0, s, false
	}
	value, ok := atoiFixed(s[:n])
	if !ok {
		return 0, s, false
	}
	return value, s[n:], true
}

func takeLiteral(s *string, c byte) bool {
	if *s == "" || (*s)[0] != c {
		return false
	}
	*s = (*s)[1:]
	return true
}

// takeLiteralFold is takeLiteral for a letter, matching either case because
// Python compiles strptime formats with re.IGNORECASE.
func takeLiteralFold(s *string, c byte) bool {
	if *s == "" || (*s)[0]|0x20 != c|0x20 {
		return false
	}
	*s = (*s)[1:]
	return true
}

// atoiFixed reads an all-ASCII-digit string. Not strconv.Atoi: that accepts a
// sign and this must not, because every caller is reading a fixed-width field
// out of a timestamp where "+1" is a different token.
func atoiFixed(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	value := 0
	for i := 0; i < len(s); i++ {
		if !isASCIIDigit(s[i]) {
			return 0, false
		}
		value = value*10 + int(s[i]-'0')
	}
	return value, true
}

// validYMD is the ordinary calendar check, used by the strptime leg.
func validYMD(year, month, day int) bool {
	return day >= 1 && validYMDAllowingDayZero(year, month, day)
}

// validYMDAllowingDayZero is the same check with the day's lower bound dropped,
// used while parsing an ISO date before the hour is known. Day 0 is legal only
// alongside hour 24; parseISOFormat rejects it otherwise.
func validYMDAllowingDayZero(year, month, day int) bool {
	if month < 1 || month > 12 || day < 0 || year < 1 || year > 9999 {
		return false
	}
	return day <= daysInMonth(year, month)
}

func daysInMonth(year, month int) int {
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
	return 31
}
