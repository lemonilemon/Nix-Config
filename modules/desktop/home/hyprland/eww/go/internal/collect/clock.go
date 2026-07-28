package collect

// Clock is collectors.clock_state's shape.
type Clock struct {
	Time    string `json:"time"`
	Date    string `json:"date"`
	Tooltip string `json:"tooltip"`
}

const (
	glyphClock    = "\uF017" // U+F017 clock
	glyphCalendar = "\uF073" // U+F073 calendar -- used by BOTH date and tooltip
)

// ClockState mirrors collectors.clock_state.
//
// The weekday name and the zone abbreviation are the two locale-sensitive
// pieces, and both are safe as Go's defaults: CPython does not call
// setlocale(LC_TIME, "") at startup and this backend never calls setlocale, so
// %A and %Z are resolved in the C locale whatever LC_TIME says. Verified
// against the original under three other locales; see period.go, which carries
// the same note for the month names.
//
// The tooltip's glyph is the CALENDAR, not the clock -- the same one the date
// field uses. That reads like a copy-paste slip and is faithful to what ships.
func ClockState() Clock {
	now := timeNow()
	return Clock{
		Time:    glyphClock + " " + now.Format("15:04"),
		Date:    glyphCalendar + " " + now.Format("2006-01-02"),
		Tooltip: glyphCalendar + " " + now.Format("Monday, 2006-01-02 MST"),
	}
}
