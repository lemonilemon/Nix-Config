package collect

// Glyphs that belong in the *Default values below, written as escapes.
//
// Every one renders as nothing -- a blank, or a box -- in an editor, a terminal
// and a diff, so a port that transcribed what it saw produced a struct whose zero
// value was silently wrong while every "checks the default" test passed. Twelve
// fields across five defaults have this shape.
//
// Take the value from here, never from a copy-paste.
const (
	GlyphMediaDefaultIcon = "\uF001"     // U+F001  MEDIA_DEFAULT["icon"]
	GlyphBatteryDefault   = "\U000F0084" // U+F0084  BATTERY_DEFAULT["text"] and ["alt"]
	GlyphBatteryPower     = "\u2014"     // U+2014  BATTERY_DEFAULT["power"], an em dash
	GlyphClockTime        = "\uF017"     // U+F017  CLOCK_DEFAULT["time"] prefix
	GlyphClockDate        = "\uF073"     // U+F073  CLOCK_DEFAULT["date"] prefix
	GlyphAiUsageDefault   = "\U000F0674" // U+F0674  AI_USAGE_DEFAULT["text"] prefix
	GlyphWorkspaceEmpty   = "\uF10C"     // U+F10C  WORKSPACE_DEFAULT["wsN_text"]
)

// MediaDefault is deliberately a function, not a package-level var: a shared Go
// value that a caller mutated would corrupt every later reader.
func MediaDefault() MediaState {
	return MediaState{Icon: GlyphMediaDefaultIcon}
}
