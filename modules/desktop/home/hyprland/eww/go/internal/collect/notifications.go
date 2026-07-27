package collect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	appMax     = 20
	summaryMax = 48
	bodyMax    = 64
)

// HistoryItem is one parsed entry of `dunstctl history`.
type HistoryItem struct {
	ID        int    `json:"id"`
	App       string `json:"app"`
	Summary   string `json:"summary"`
	Body      string `json:"body"`
	Urgency   string `json:"urgency"`
	Timestamp int64  `json:"timestamp"`
}

// NotificationItem is one entry as the popup renders it.
type NotificationItem struct {
	ID      int    `json:"id"`
	Summary string `json:"summary"`
	Body    string `json:"body"`
	Age     string `json:"age"`
	Urgency string `json:"urgency"`
}

// NotificationGroup is one app's notifications.
type NotificationGroup struct {
	App       string             `json:"app"`
	Count     int                `json:"count"`
	Collapsed string             `json:"collapsed"`
	Items     []NotificationItem `json:"items"`
}

// NotificationsState is the shape notifications_state_from_parts returns.
type NotificationsState struct {
	Paused string              `json:"paused"`
	New    int                 `json:"new"`
	Count  int                 `json:"count"`
	Groups []NotificationGroup `json:"groups"`
}

// pyStr renders a decoded JSON value the way Python's str() would.
//
// The distinction Go loses by default is int versus float: json.loads gives 5
// for "5" and 5.0 for "5.0", and str() of those is "5" and "5.0". Decoding into
// float64 collapses both to 5, so the history is decoded with UseNumber and the
// original literal is kept. Booleans are Python-cased, though False never
// reaches here -- the caller's `or` treats it as falsy first.
func pyStr(value any) string {
	switch typed := value.(type) {
	case nil:
		return "None"
	case string:
		return typed
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

// pyTruthy is Python's truthiness for the values json.Unmarshal produces.
func pyTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return typed != ""
	case bool:
		return typed
	case json.Number:
		if f, err := typed.Float64(); err == nil {
			return f != 0
		}
		return typed.String() != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	}
	return true
}

// dunstField mirrors notifications._field.
//
// Note what it does NOT do: if the value is not a {"type":..,"data":..} wrapper
// it returns the default, not the value. A bare `"appname": "kitty"` therefore
// reads as absent. That is the original's behaviour and dunst always wraps.
func dunstField(item map[string]any, name string, def any) any {
	value, present := item[name]
	if !present {
		return def
	}
	wrapper, isWrapper := value.(map[string]any)
	if !isWrapper {
		return def
	}
	inner, present := wrapper["data"]
	if !present {
		return def
	}
	return inner
}

// toInt is Python's int() over the values that reach it here: it truncates
// floats toward zero and parses integral strings, and reports failure where
// Python would raise ValueError (which the caller turns into `continue`).
func toInt(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return i, true
		}
		f, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return int64(math.Trunc(f)), true
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return i, true
	case bool:
		if typed {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// ParseHistoryItems mirrors notifications.parse_history_items.
//
// `dunstctl history` wraps everything in {"type": "aa{sv}", "data": [[...]]}
// and every field in {"type": ..., "data": ...}. timestamp is MICROSECONDS on
// the boot clock, not wall time.
func ParseHistoryItems(historyJSON string) []HistoryItem {
	decoder := json.NewDecoder(bytes.NewReader([]byte(historyJSON)))
	decoder.UseNumber() // keep 5 distinct from 5.0, see pyStr
	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		return []HistoryItem{}
	}

	data, _ := body["data"].([]any)
	if len(data) == 0 {
		return []HistoryItem{}
	}
	first, ok := data[0].([]any)
	if !ok {
		return []HistoryItem{}
	}

	items := []HistoryItem{}
	for _, raw := range first {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, ok := toInt(dunstField(entry, "id", json.Number("0")))
		if !ok {
			continue
		}
		timestamp, ok := toInt(dunstField(entry, "timestamp", json.Number("0")))
		if !ok {
			continue
		}
		items = append(items, HistoryItem{
			ID:        int(id),
			App:       orDefault(dunstField(entry, "appname", ""), "unknown"),
			Summary:   orDefault(dunstField(entry, "summary", ""), ""),
			Body:      orDefault(dunstField(entry, "body", ""), ""),
			Urgency:   orDefault(dunstField(entry, "urgency", ""), "NORMAL"),
			Timestamp: timestamp,
		})
	}

	// SliceStable, not Slice: Python's sort is stable, and dunst can stamp two
	// notifications with the same microsecond. An unstable sort would reorder
	// them arbitrarily between runs, which the emit loop would see as a change
	// and push to eww for no reason.
	sort.SliceStable(items, func(a, b int) bool {
		return items[a].Timestamp > items[b].Timestamp
	})
	return items
}

// orDefault is Python's `str(x or fallback)`.
func orDefault(value any, fallback string) string {
	if !pyTruthy(value) {
		return fallback
	}
	return pyStr(value)
}

// FormatAge mirrors notifications.format_age.
func FormatAge(seconds float64) string {
	switch {
	case seconds < 10:
		return "now"
	case seconds < 60:
		return fmt.Sprintf("%ds", int64(seconds))
	case seconds < 3600:
		return fmt.Sprintf("%dm", int64(math.Floor(seconds/60)))
	case seconds < 86400:
		return fmt.Sprintf("%dh", int64(math.Floor(seconds/3600)))
	default:
		return fmt.Sprintf("%dd", int64(math.Floor(seconds/86400)))
	}
}

// NotificationsStateFromParts mirrors notifications.notifications_state_from_parts.
func NotificationsStateFromParts(
	items []HistoryItem,
	pausedText string,
	nowMonotonicUS float64,
	collapsed map[string]bool,
	lastSeenUS int64,
) NotificationsState {
	// grouped + order, not a plain map: Go map iteration is randomised and the
	// original relies on first-appearance order, which is what puts the most
	// recent app at the top of the popup.
	grouped := map[string][]NotificationItem{}
	var order []string

	for _, item := range items {
		app := TruncateText(item.App, appMax)
		if _, seen := grouped[app]; !seen {
			grouped[app] = []NotificationItem{}
			order = append(order, app)
		}
		// NOT covered by the equivalence gate, and cannot be: FormatAge returns
		// "now" for everything below 10, so clamping -8 to 0 changes no output
		// anyone can observe. The clamp is defensive against a future threshold
		// change, and removing it would pass every test here. Kept because the
		// original has it and a negative age is reachable -- dunst stamps
		// CLOCK_BOOTTIME, and a notification can be newer than the reading the
		// caller took a moment earlier.
		age := math.Max(0, (nowMonotonicUS-float64(item.Timestamp))/1_000_000)
		grouped[app] = append(grouped[app], NotificationItem{
			ID:      item.ID,
			Summary: TruncateText(item.Summary, summaryMax),
			Body:    TruncateText(strings.ReplaceAll(item.Body, "\n", " "), bodyMax),
			Age:     FormatAge(age),
			Urgency: item.Urgency,
		})
	}

	groups := []NotificationGroup{}
	for _, app := range order {
		collapsedFlag := "false"
		if collapsed[app] {
			collapsedFlag = "true"
		}
		groups = append(groups, NotificationGroup{
			App:       app,
			Count:     len(grouped[app]),
			Collapsed: collapsedFlag,
			Items:     grouped[app],
		})
	}

	newCount := 0
	for _, item := range items {
		if item.Timestamp > lastSeenUS {
			newCount++
		}
	}

	paused := "false"
	if Strip(pausedText) == "true" {
		paused = "true"
	}
	return NotificationsState{Paused: paused, New: newCount, Count: len(items), Groups: groups}
}
