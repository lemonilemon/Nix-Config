package collect

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// The quota cards in the AI popup. Two providers feed them, with different
// shapes: openusage-cli emits a generic "lines" report, and the Claude usage
// endpoint emits named window objects. Both land in the same Quota.

// QuotaWindow is one usage window on a provider card, in emit order.
type QuotaWindow struct {
	Label     string `json:"label"`
	Percent   int    `json:"percent"`
	Value     string `json:"value"`
	Remaining string `json:"remaining"`
	Reset     string `json:"reset"`
	Class     string `json:"class"`
}

// QuotaMeta is one non-percentage row, such as a credit balance.
type QuotaMeta struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

// Quota is one provider card.
//
// Windows and Meta carry concrete types. An earlier draft of this port used
// []any on the theory that typing them meant re-deriving six third-party
// schemas in Go -- which was wrong. The Python derives no schemas; it probes
// alternative key names on untyped dicts and normalises into exactly these two
// fixed shapes, which is all that ever reaches eww.
type Quota struct {
	Key     string        `json:"key"`
	Name    string        `json:"name"`
	Plan    string        `json:"plan"`
	Status  string        `json:"status"`
	Class   string        `json:"class"`
	Updated string        `json:"updated"`
	Windows []QuotaWindow `json:"windows"`
	Meta    []QuotaMeta   `json:"meta"`
}

// QuotaDefaults is AI_USAGE_DEFAULT["quotas"]: the three cards the popup shows
// before any probe has run.
//
// A function rather than a var because the Python deep-copies the template at
// every use, and a shared Go value a caller mutated would corrupt every later
// reader.
func QuotaDefaults() []Quota {
	blank := func(key, name string) Quota {
		return Quota{
			Key: key, Name: name, Plan: "--", Status: "waiting",
			Class: "missing", Updated: "",
			Windows: []QuotaWindow{}, Meta: []QuotaMeta{},
		}
	}
	return []Quota{
		blank("claude", "Claude"),
		blank("codex", "Codex"),
		blank("antigravity", "Antigravity"),
	}
}

// QuotaDefault mirrors collectors.quota_default: a blank card for one provider,
// carrying a status.
//
// The name argument is only a fallback. For a key that has a default card, the
// NAME COMES FROM THE TEMPLATE -- quota_default("claude", "whatever") is still
// "Claude" -- and only an unrecognised key uses what the caller passed.
func QuotaDefault(key, name, status string) Quota {
	for _, card := range QuotaDefaults() {
		if card.Key == key {
			card.Status = status
			return card
		}
	}
	return Quota{
		Key: key, Name: name, Plan: "--", Status: status,
		Class: "missing", Updated: "",
		Windows: []QuotaWindow{}, Meta: []QuotaMeta{},
	}
}

// ClaudeQuotaDefault mirrors collectors.claude_quota_default.
func ClaudeQuotaDefault(status string) Quota { return QuotaDefault("claude", "Claude", status) }

// QuotaWindowClass mirrors collectors.quota_window_class.
//
// "empty" and "missing" both mean zero usage; they differ in whether a reset
// time was known, which is how the popup tells "nothing used yet this window"
// from "no data at all".
func QuotaWindowClass(percent int, hasReset bool) string {
	switch {
	case percent >= 90:
		return "critical"
	case percent >= 80:
		return "warning"
	case percent > 0:
		return "active"
	case hasReset:
		return "empty"
	}
	return "missing"
}

// QuotaCardClass mirrors collectors.quota_card_class: the card takes the most
// severe class any of its rows carries.
func QuotaCardClass(quota Quota) string {
	present := map[string]bool{}
	for _, window := range quota.Windows {
		present[window.Class] = true
	}
	for _, item := range quota.Meta {
		present[item.Class] = true
	}
	for _, name := range [...]string{"critical", "warning", "active", "empty"} {
		if present[name] {
			return name
		}
	}
	return "missing"
}

// timeNow is the clock seam. Package-level var for the same reason as RunText:
// a test swaps it and restores it on defer, and InstallFixture drives it from
// the "now" key so a recorded case resolves the same wall clock every time.
var timeNow = time.Now

// nowOr mirrors the `now_epoch or time.time()` idiom these collectors all open
// with. It is `or`, not a None check, so a caller passing 0 gets the wall clock
// rather than the epoch -- reproduced rather than corrected.
func nowOr(nowEpoch float64) float64 {
	if nowEpoch != 0 {
		return nowEpoch
	}
	return float64(timeNow().UnixNano()) / 1e9
}

// clockHHMM is time.strftime("%H:%M", time.localtime(now)).
func clockHHMM(now float64) string {
	return time.Unix(int64(now), 0).Local().Format("15:04")
}

// OpenusageWindow mirrors collectors.openusage_window.
func OpenusageWindow(line map[string]any, nowEpoch float64) QuotaWindow {
	now := nowOr(nowEpoch)
	percent := PercentPart(NumberValue(line, "used"), NumberValue(line, "limit"))

	resetEpoch, parsed := ParseISOEpoch(line["resetsAt"])
	hasReset := parsed && resetEpoch > 0

	remaining := "--"
	if hasReset && resetEpoch > now {
		remaining = FormatRemaining(max(0, resetEpoch-now))
	}
	reset := "--"
	if hasReset {
		reset = FormatClockTime(resetEpoch)
	}

	return QuotaWindow{
		Label:     TruncateText(orDefault(line["label"], "?"), 22),
		Percent:   percent,
		Value:     strconv.Itoa(percent) + "%",
		Remaining: remaining,
		Reset:     reset,
		Class:     QuotaWindowClass(percent, hasReset),
	}
}

// pyOr is Python's `x or fallback` without the str() that orDefault applies:
// the Claude reset lookup feeds the result to ParseISOEpoch, which treats a
// non-string as unparseable, so stringifying here would change the answer.
func pyOr(value, fallback any) any {
	if pyTruthy(value) {
		return value
	}
	return fallback
}

// OpenusageMeta mirrors collectors.openusage_meta, reporting ok=false where the
// original returns None.
//
// Count lines are used/limit pairs; what is worth glancing at is what remains,
// so an exhausted balance is hidden rather than shown as "0 credits".
func OpenusageMeta(line map[string]any) (QuotaMeta, bool) {
	format, _ := line["format"].(map[string]any)

	// int() twice, then subtract -- not one subtraction then int(). Both
	// operands truncate toward zero first, so 1.9 - 0.9 is 1, not 1.0.
	remaining := int64(NumberValue(line, "limit")) - int64(NumberValue(line, "used"))
	if remaining <= 0 {
		return QuotaMeta{}, false
	}

	value := strconv.FormatInt(remaining, 10)
	if suffix, isText := format["suffix"].(string); isText && suffix != "" {
		value += " " + suffix
	}
	return QuotaMeta{
		Label:   orDefault(line["label"], "?"),
		Value:   value,
		Tooltip: "",
		Class:   "active",
	}, true
}

// maxQuotaWindows caps how many windows a card shows. Every window is a heavy
// widget, so the ones nearest their limit are kept and the card stays short.
const maxQuotaWindows = 4

// QuotaFromOpenusage mirrors collectors.quota_from_openusage.
//
// snapshot is `any` because the caller looks it up by provider id in a map
// built from a report that may not contain it; a non-map value, including nil,
// is the "unavailable" path.
func QuotaFromOpenusage(snapshot any, key, name string, nowEpoch float64) Quota {
	now := nowOr(nowEpoch)

	report, isMap := snapshot.(map[string]any)
	if !isMap {
		return QuotaDefault(key, name, "unavailable")
	}

	var lines []map[string]any
	for _, raw := range ListValue(report, "lines") {
		if line, isLine := raw.(map[string]any); isLine {
			lines = append(lines, line)
		}
	}

	// A probe failure comes back as a single text line labelled "Error".
	for _, line := range lines {
		if line["type"] == "text" && line["label"] == "Error" {
			status := TruncateText(orDefault(line["value"], "error"), 48)
			return QuotaDefault(key, name, status)
		}
	}

	windows := []QuotaWindow{}
	meta := []QuotaMeta{}
	for _, line := range lines {
		if line["type"] != "progress" {
			continue
		}
		format, _ := line["format"].(map[string]any)
		if format["kind"] == "percent" {
			windows = append(windows, OpenusageWindow(line, now))
			continue
		}
		if entry, ok := OpenusageMeta(line); ok {
			meta = append(meta, entry)
		}
	}
	if len(windows) == 0 && len(meta) == 0 {
		return QuotaDefault(key, name, "missing")
	}

	if len(windows) > maxQuotaWindows {
		// Stable, and sorted only when the cap actually bites -- under it the
		// original leaves report order alone, which is not the same list.
		sort.SliceStable(windows, func(a, b int) bool {
			if windows[a].Percent != windows[b].Percent {
				return windows[a].Percent > windows[b].Percent
			}
			return PyLower(windows[a].Label) < PyLower(windows[b].Label)
		})
		windows = windows[:maxQuotaWindows]
	}

	quota := QuotaDefault(key, name, "live")
	quota.Updated = clockHHMM(now)
	if plan, isText := report["plan"].(string); isText && Strip(plan) != "" {
		quota.Plan = Strip(plan)
	}
	quota.Windows = windows
	quota.Meta = meta
	quota.Class = QuotaCardClass(quota)
	return quota
}

// FormatClaudePlan mirrors collectors.format_claude_plan.
func FormatClaudePlan(value any) string {
	text, isText := value.(string)
	if !isText || Strip(text) == "" {
		return "--"
	}
	return PyTitle(strings.ReplaceAll(Strip(text), "_", " "))
}

// claudeWindowKeys is the label and the key spellings for each Claude usage
// window, in the order the card renders them.
var claudeWindowKeys = []struct {
	Label string
	Keys  []string
}{
	{"Session", []string{"five_hour", "fiveHour"}},
	{"Weekly", []string{"seven_day", "sevenDay"}},
	{"Opus weekly", []string{"seven_day_opus", "sevenDayOpus"}},
	{"Sonnet weekly", []string{"seven_day_sonnet", "sevenDaySonnet"}},
}

// ClaudeWindowState mirrors collectors.claude_window_state.
func ClaudeWindowState(label string, window any, nowEpoch float64) QuotaWindow {
	now := nowOr(nowEpoch)

	fields, isMap := window.(map[string]any)
	if !isMap {
		return QuotaWindow{
			Label: label, Percent: 0, Value: "--",
			Remaining: "--", Reset: "--", Class: "missing",
		}
	}

	percent := ClampPercent(NumberValue(fields, "utilization", "used_percent", "percent"))

	// The reset is an ISO string in the current API and was a bare epoch in an
	// older one, so a failed parse falls back to reading it as a number.
	resetEpoch, parsed := ParseISOEpoch(pyOr(fields["resets_at"], fields["reset_at"]))
	if !parsed {
		resetEpoch = NumberValue(fields, "resets_at", "reset_at")
	}

	remaining, reset := "--", "--"
	if resetEpoch > 0 {
		remaining = FormatRemaining(max(0, resetEpoch-now))
		reset = FormatClockTime(resetEpoch)
	}

	return QuotaWindow{
		Label:     label,
		Percent:   percent,
		Value:     strconv.Itoa(percent) + "%",
		Remaining: remaining,
		Reset:     reset,
		Class:     QuotaWindowClass(percent, resetEpoch > 0),
	}
}

// ClaudeQuotaStateFromJSON mirrors collectors.claude_quota_state_from_json.
func ClaudeQuotaStateFromJSON(usageJSON string, plan any, nowEpoch float64) Quota {
	var body map[string]any
	if !ParseJSON(usageJSON, &body) || len(body) == 0 {
		return ClaudeQuotaDefault("missing")
	}
	now := nowOr(nowEpoch)

	windows := []QuotaWindow{}
	for _, spec := range claudeWindowKeys {
		for _, key := range spec.Keys {
			if fields, isMap := body[key].(map[string]any); isMap {
				windows = append(windows, ClaudeWindowState(spec.Label, fields, now))
				break
			}
		}
	}
	if len(windows) == 0 {
		return ClaudeQuotaDefault("unrecognized data")
	}

	quota := ClaudeQuotaDefault("live")
	quota.Updated = clockHHMM(now)
	quota.Plan = FormatClaudePlan(plan)
	quota.Windows = windows
	quota.Class = QuotaCardClass(quota)
	return quota
}
