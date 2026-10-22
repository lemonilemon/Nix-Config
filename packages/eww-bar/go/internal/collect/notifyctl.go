package collect

import (
	"strconv"
	"sync"
	"time"
)

// The impure half of notifications: reading dunst's history, the six actions
// the popup binds, and the process-lifetime UI state behind them.

const dunstctlTimeout = 5 * time.Second

// Ephemeral UI state, process lifetime only: collapse and badge state is fine to
// lose on a daemon restart, which is why it is memory rather than a runtime file.
var (
	notifyUILock  sync.Mutex
	collapsedApps = map[string]bool{}
	lastSeenUS    int64
)

// ResetNotifyUIState drops the collapse and badge state. For tests.
func ResetNotifyUIState() {
	notifyUILock.Lock()
	defer notifyUILock.Unlock()
	collapsedApps = map[string]bool{}
	lastSeenUS = 0
}

// BoottimeSeconds: dunst stamps history with CLOCK_BOOTTIME, which INCLUDES
// suspend; CLOCK_MONOTONIC instead makes ages go negative after any suspend.
//
// Go's standard library exposes no clock_gettime and this module has no dependencies
// to add one, so the value comes from /proc/uptime, the same clock at centisecond
// resolution.
var BoottimeSeconds = func() float64 {
	text, ok := ReadTextFile("/proc/uptime")
	if !ok {
		return 0.0
	}
	fields := SplitWhitespaceN(text, -1)
	if len(fields) == 0 {
		return 0.0
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0.0
	}
	return seconds
}

func dunstHistoryItems() []HistoryItem {
	return ParseHistoryItems(RunText(defaultTimeout, "dunstctl", "history"))
}

// CollectNotifications is named for the package's convention: pure parsers are
// XFromParts with plain-noun results, so the impure wrapper is CollectX.
func CollectNotifications() NotificationsState {
	items := dunstHistoryItems()
	pausedText := RunText(defaultTimeout, "dunstctl", "is-paused")

	notifyUILock.Lock()
	collapsed := map[string]bool{}
	for app := range collapsedApps {
		collapsed[app] = true
	}
	lastSeen := lastSeenUS
	notifyUILock.Unlock()

	return NotificationsStateFromParts(
		items, pausedText, BoottimeSeconds()*1_000_000, collapsed, lastSeen)
}

func runDunstctl(args ...string) {
	RunStatus(dunstctlTimeout, "dunstctl", args...)
}

func ToggleGroup(app string) (NotificationsState, error) {
	if app == "" {
		return NotificationsState{}, errValue("notif toggle-group requires an app name")
	}
	notifyUILock.Lock()
	if collapsedApps[app] {
		delete(collapsedApps, app)
	} else {
		collapsedApps[app] = true
	}
	notifyUILock.Unlock()
	return CollectNotifications(), nil
}

func DismissNotification(id any) (NotificationsState, error) {
	value, ok := pyInt(id)
	if !ok {
		return NotificationsState{}, errValue("notif dismiss requires a numeric id")
	}
	runDunstctl("history-rm", strconv.FormatInt(value, 10))
	return CollectNotifications(), nil
}

func ClearGroup(app string) (NotificationsState, error) {
	if app == "" {
		return NotificationsState{}, errValue("notif clear-group requires an app name")
	}
	// Group names are the TRUNCATED app names, so the match has to truncate
	// the same way or a long app name would never clear.
	for _, item := range dunstHistoryItems() {
		if TruncateText(item.App, appMax) == app {
			runDunstctl("history-rm", strconv.FormatInt(int64(item.ID), 10))
		}
	}
	return CollectNotifications(), nil
}

func ClearAllNotifications() NotificationsState {
	runDunstctl("history-clear")
	return CollectNotifications()
}

func ToggleDND() NotificationsState {
	runDunstctl("set-paused", "toggle")
	return CollectNotifications()
}

// MarkSeen: the badge counts what arrived after the last time the popup was opened.
func MarkSeen() NotificationsState {
	items := dunstHistoryItems()
	newest := int64(BoottimeSeconds() * 1_000_000)
	if len(items) > 0 {
		newest = items[0].Timestamp
	}
	notifyUILock.Lock()
	if newest > lastSeenUS {
		lastSeenUS = newest
	}
	notifyUILock.Unlock()
	return CollectNotifications()
}

// pyInt is int(value) for the shapes a control payload can carry: the JSON
// decoder gives a string for `notif dismiss <id>`, and int() accepts a numeric
// string, a number, or a bool. A float truncates toward zero.
func pyInt(value any) (int64, bool) {
	switch typed := value.(type) {
	case string:
		parsed, err := strconv.ParseInt(Strip(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	return toInt(value)
}

// errValue is Python's ValueError as this package reports it: the control
// handler turns these into {"ok": false, "error": <message>} and the client
// prints them, so the text is part of the interface.
func errValue(message string) error { return valueError(message) }

type valueError string

func (e valueError) Error() string { return string(e) }
