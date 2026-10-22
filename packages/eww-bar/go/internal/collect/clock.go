package collect

type Clock struct {
	Time    string `json:"time"`
	Date    string `json:"date"`
	Tooltip string `json:"tooltip"`
}

const (
	glyphClock    = "\uF017" // U+F017 clock
	glyphCalendar = "\uF073" // U+F073 calendar -- used by BOTH date and tooltip
)

// ClockState resolves the weekday name and zone abbreviation in the C locale, as
// the original did: neither side calls setlocale, so %A and %Z ignore LC_TIME.
//
// The tooltip's glyph is the CALENDAR, not the clock. Faithful to what ships.
func ClockState() Clock {
	now := timeNow()
	return Clock{
		Time:    glyphClock + " " + now.Format("15:04"),
		Date:    glyphCalendar + " " + now.Format("2006-01-02"),
		Tooltip: glyphCalendar + " " + now.Format("Monday, 2006-01-02 MST"),
	}
}
