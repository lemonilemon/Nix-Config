package collect

import (
	"fmt"
	"regexp"
	"strings"
)

// The Nerd Font glyphs the volume label carries. Written as escapes, not
// literals: they are Private Use Area code points that render as blanks or
// boxes in most editors and diffs, so a literal here is one careless
// copy-paste away from silently becoming a different glyph. Lifted from
// collectors.volume_label_from_text by code point, not by eye.
const (
	glyphVolumeMuted = "\U000F075F" // U+F075F mute
	glyphVolumeLow   = "\uF026"     // U+F026 speaker, low
	glyphVolumeMid   = "\uF027"     // U+F027 speaker, medium
	glyphVolumeHigh  = "\uF028"     // U+F028 speaker, high
)

var volumeRe = regexp.MustCompile(`Volume:\s+([0-9.]+)`)

// VolumeState is the shape collectors.volume_state_from_text returns.
//
// A struct rather than a map: encoding/json sorts map keys but emits struct
// fields in declaration order, and the order below is the order the Python dict
// is built in, which is the order that reaches eww.
type VolumeState struct {
	Text    string `json:"text"`
	Percent int    `json:"percent"`
	Muted   string `json:"muted"`
	Class   string `json:"class"`
	Sinks   []Sink `json:"sinks"`
}

// Sink is one entry of the volume popup's output-device list.
type Sink struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      string `json:"active"`
}

// VolumeLabelFromText mirrors collectors.volume_label_from_text.
func VolumeLabelFromText(text string) string {
	match := volumeRe.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	volume, ok := DecimalTimes100HalfUp(match[1])
	if !ok {
		// Unreachable from wpctl, and the original does not handle it: Decimal()
		// raises on something like "1.2.3", which the [0-9.]+ class does admit.
		// Degrade to the no-reading answer rather than propagating, which is
		// where the exception would have landed anyway.
		return ""
	}
	if strings.Contains(text, "[MUTED]") {
		return glyphVolumeMuted
	}
	switch {
	case volume < 35:
		return fmt.Sprintf("%s %d%%", glyphVolumeLow, volume)
	case volume < 70:
		return fmt.Sprintf("%s %d%%", glyphVolumeMid, volume)
	default:
		return fmt.Sprintf("%s %d%%", glyphVolumeHigh, volume)
	}
}

// VolumeStateFromText mirrors collectors.volume_state_from_text.
func VolumeStateFromText(text string, sinks []Sink) VolumeState {
	if sinks == nil {
		// Not cosmetic: the original's `sinks or []` means this field is never
		// null, and eww.yuck runs (for sink in ...) over it. A nil slice
		// marshals to null, which eww cannot index.
		sinks = []Sink{}
	}
	label := VolumeLabelFromText(text)

	match := volumeRe.FindStringSubmatch(text)
	if match == nil {
		return VolumeState{Text: "", Percent: 0, Muted: "false", Class: "", Sinks: sinks}
	}
	percent, ok := DecimalTimes100HalfUp(match[1])
	if !ok {
		return VolumeState{Text: "", Percent: 0, Muted: "false", Class: "", Sinks: sinks}
	}

	muted := strings.Contains(text, "[MUTED]")
	state := VolumeState{Text: label, Percent: percent, Muted: "false", Sinks: sinks}
	if muted {
		state.Muted = "true"
		state.Class = "muted"
	}
	return state
}

// VolumeEventIsRelevant mirrors collectors.volume_event_is_relevant.
func VolumeEventIsRelevant(line string) bool {
	return !strings.Contains(line, " on client #")
}

// SinksFromPactlJSON mirrors collectors.sinks_from_pactl_json.
func SinksFromPactlJSON(jsonText, defaultName string) []Sink {
	var data []any
	sinks := []Sink{}
	if !ParseJSON(jsonText, &data) {
		return sinks
	}
	for _, raw := range data {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, ok := entry["name"].(string)
		if !ok || name == "" {
			continue
		}
		label := name
		if description, ok := entry["description"].(string); ok && description != "" {
			label = description
		}
		sinks = append(sinks, Sink{
			Name:        name,
			Description: TruncateText(label, 30),
			Active:      boolText(name == defaultName),
		})
	}
	return sinks
}

// SinksFromPactlShort mirrors collectors.sinks_from_pactl_short.
//
// Splits on tab, not whitespace: pactl's short format is tab-separated and a
// sink description can contain spaces.
func SinksFromPactlShort(shortText, defaultName string) []Sink {
	sinks := []Sink{}
	for _, line := range SplitLines(shortText) {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 || fields[1] == "" {
			continue
		}
		name := fields[1]
		sinks = append(sinks, Sink{
			Name:        name,
			Description: TruncateText(name, 30),
			Active:      boolText(name == defaultName),
		})
	}
	return sinks
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
