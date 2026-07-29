package collect

// Glyphs that belong in the *Default values below, written as escapes.
//
// Every one of these renders as nothing -- a blank, or a box -- in an editor, a
// terminal and a diff, so in the original Python source they looked exactly like
// "". A port that read that source and transcribed what it saw produced a struct
// whose zero value was silently wrong, and every test that only checked "the
// default" passed.
//
// This is not hypothetical: MediaDefault below was written as an empty struct,
// the equivalence gate rejected it, and the reason was this U+F001. Twelve
// fields across five defaults have the same shape -- MEDIA_DEFAULT.icon,
// BATTERY_DEFAULT.text/.alt/.power, CLOCK_DEFAULT.time/.date,
// AI_USAGE_DEFAULT.text, and WORKSPACE_DEFAULT.ws1..5_text.
//
// Take the value from here, never from a copy-paste of the Python.
const (
	GlyphMediaDefaultIcon = "\uF001"     // U+F001  MEDIA_DEFAULT["icon"]
	GlyphBatteryDefault   = "\U000F0084" // U+F0084  BATTERY_DEFAULT["text"] and ["alt"]
	GlyphBatteryPower     = "\u2014"     // U+2014  BATTERY_DEFAULT["power"], an em dash
	GlyphClockTime        = "\uF017"     // U+F017  CLOCK_DEFAULT["time"] prefix
	GlyphClockDate        = "\uF073"     // U+F073  CLOCK_DEFAULT["date"] prefix
	GlyphAiUsageDefault   = "\U000F0674" // U+F0674  AI_USAGE_DEFAULT["text"] prefix
	GlyphWorkspaceEmpty   = "\uF10C"     // U+F10C  WORKSPACE_DEFAULT["wsN_text"]
)

// MediaDefault is common.MEDIA_DEFAULT.
//
// Deliberately a function, not a package-level var: the Python is copied at
// every use (MEDIA_DEFAULT.copy()), and a shared Go value that a caller mutated
// would corrupt every later reader.
func MediaDefault() MediaState {
	return MediaState{Icon: GlyphMediaDefaultIcon}
}
