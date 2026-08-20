// Package control is the daemon side of the control socket: the command
// dispatch behind eww-barctl, and the server that carries it.
//
// Its own package rather than part of collect, because the handler writes into
// BarState and package state already imports collect -- the dependency cannot
// run both ways.
package control

import (
	"errors"
	"sync"

	"ewwbar/internal/collect"
	"ewwbar/internal/paths"
	"ewwbar/internal/pyjson"
	"ewwbar/internal/state"
)

// Reply is one control response, in key order. Ordered rather than a map
// because the client echoes these bytes verbatim and eww.yuck reads them; a Go
// map would sort the keys and change the wire format.
type Reply = pyjson.Ordered

func ok(command string, rest ...pyjson.Pair) Reply {
	return append(Reply{{Key: "ok", Value: true}, {Key: "command", Value: command}}, rest...)
}

// ErrorReply is the shape every failure takes, including a bad payload.
func ErrorReply(message string) Reply {
	return Reply{{Key: "ok", Value: false}, {Key: "error", Value: message}}
}

func pair(key string, value any) pyjson.Pair { return pyjson.Pair{Key: key, Value: value} }

var aiRefreshLock sync.Mutex
var aiRefreshBusy bool

// QueueAiRefresh mirrors control.queue_ai_refresh.
//
// Non-blocking: a second refresh while one is running is DROPPED rather than
// queued, and reported as "already-refreshing". The popup sends one of these
// every time it opens, and the underlying probe can take 40 s, so queueing
// would let a few opens pile up into minutes of redundant work.
func QueueAiRefresh(store *state.Store) bool {
	aiRefreshLock.Lock()
	if aiRefreshBusy {
		aiRefreshLock.Unlock()
		return false
	}
	aiRefreshBusy = true
	aiRefreshLock.Unlock()

	go func() {
		defer func() {
			aiRefreshLock.Lock()
			aiRefreshBusy = false
			aiRefreshLock.Unlock()
		}()
		collect.RefreshAiUsage(store.Get().AiUsage, func(value collect.AiUsage) {
			store.Update(func(bar *state.Bar) { bar.AiUsage = value })
		}, true)
	}()
	return true
}

var speedtestLock sync.Mutex
var speedtestBusy bool

// QueueSpeedtest starts one measurement, or reports that one is already going.
//
// Same drop-don't-queue shape as QueueAiRefresh, for a sharper reason: this
// moves about 70 MB and saturates the link for nine seconds, so a double-click
// that queued a second run would spend the data twice and report the first
// run's numbers measured against the second run's congestion.
//
// The publish closure updates only bar.Network.Speed rather than re-running
// CollectNetwork. That keeps the four state changes of a single run down to no
// forks at all: the identity and connectivity already in the state are what the
// card needs, and re-collecting them would fork nmcli four more times to learn
// what has not changed.
func QueueSpeedtest(store *state.Store) bool {
	speedtestLock.Lock()
	if speedtestBusy {
		speedtestLock.Unlock()
		return false
	}
	speedtestBusy = true
	speedtestLock.Unlock()

	go func() {
		defer func() {
			speedtestLock.Lock()
			speedtestBusy = false
			speedtestLock.Unlock()
		}()

		collect.RunSpeedtest(func() {
			store.Update(func(bar *state.Bar) {
				bar.Network.Speed = collect.SpeedtestCardFor(
					bar.Network.ConnUUID, bar.Network.Class,
					bar.Network.Connectivity, 0)
			})
		})

		// Persisted here rather than inside RunSpeedtest for the reason
		// SaveQuotaSnapshot is called from app.go: the collector has no
		// business knowing a path.
		_ = collect.SaveSpeedtestRecords(paths.Speedtest())
	}()
	return true
}

// payloadString reads a payload field as Python's payload.get(key, fallback)
// does: a missing key or a non-string both take the fallback, because the
// original indexes straight into the reply with it.
func payloadString(payload map[string]any, key, fallback string) string {
	if value, isText := payload[key].(string); isText {
		return value
	}
	if _, present := payload[key]; !present {
		return fallback
	}
	return fallback
}

// Handle mirrors control.handle_control_command.
//
// Returns the reply and an error; the caller turns a non-nil error into
// ErrorReply, matching serve_control_connection's except branch.
func Handle(store *state.Store, payload map[string]any) (Reply, error) {
	command := payloadString(payload, "command", "")

	switch command {
	case "ping":
		return ok("ping"), nil

	case "volume":
		action := payloadString(payload, "action", "")
		var value collect.VolumeState
		var err error
		switch action {
		case "up", "down":
			value, err = collect.AdjustVolume(action)
		case "set":
			value, err = collect.SetVolume(payload["value"])
		case "mute":
			value = collect.ToggleMute()
		case "sink":
			value, err = collect.SetSink(payloadString(payload, "sink", ""))
		default:
			err = errValue("volume action must be up, down, set, mute, or sink")
		}
		if err != nil {
			return nil, err
		}
		store.Update(func(bar *state.Bar) { bar.Volume = value })
		return ok("volume", pair("action", action), pair("volume", value)), nil

	case "media":
		action := payloadString(payload, "action", "")
		value, err := collect.ControlMedia(action)
		if err != nil {
			return nil, err
		}
		store.Update(func(bar *state.Bar) { bar.Media = value })
		return ok("media", pair("action", action), pair("media", value)), nil

	case "bluetooth":
		action := payloadString(payload, "action", "")
		var value collect.BluetoothState
		var err error
		switch action {
		case "power-toggle":
			value = collect.ToggleBluetoothPower()
		case "disconnect":
			value, err = collect.DisconnectBluetooth(payloadString(payload, "mac", ""))
		default:
			err = errValue("bluetooth action must be power-toggle or disconnect")
		}
		if err != nil {
			return nil, err
		}
		store.Update(func(bar *state.Bar) { bar.Bluetooth = value })
		return ok("bluetooth", pair("action", action), pair("bluetooth", value)), nil

	case "network":
		action := payloadString(payload, "action", "")
		switch action {
		case "wifi-toggle":
			value := collect.ToggleWifi()
			store.Update(func(bar *state.Bar) { bar.Network = value })
			return ok("network", pair("action", action), pair("network", value)), nil
		case "speedtest":
			status := "already-running"
			if QueueSpeedtest(store) {
				status = "started"
			}
			return ok("network", pair("action", action), pair("status", status)), nil
		}
		return nil, errValue("network action must be wifi-toggle or speedtest")

	case "idle":
		action := payloadString(payload, "action", "toggle")
		var value string
		var err error
		switch action {
		case "toggle":
			value, err = collect.ToggleIdleInhibited()
		case "on":
			value, err = collect.SetIdleInhibited(true)
		case "off":
			value, err = collect.SetIdleInhibited(false)
		case "status":
			value = collect.IdleInhibitedState()
		default:
			err = errValue("idle action must be toggle, on, off, or status")
		}
		if err != nil {
			return nil, err
		}
		// The display card shows the lid inhibitor, which the idle actions do
		// not touch -- but display_state is the only thing that reads it, so it
		// is refreshed here to keep the popup consistent.
		displayValue := collect.DisplayState("")
		store.Update(func(bar *state.Bar) {
			bar.IdleInhibited = value
			bar.Display = displayValue
		})
		// No "action" key, unlike every other command. Preserved because the
		// reply shape is what eww.yuck parses.
		return ok("idle", pair("idle_inhibited", value)), nil

	case "display":
		action := payloadString(payload, "action", "status")
		value, err := collect.SetDisplayMode(action)
		if err != nil {
			// Put the failure somewhere a human will see it. Every one of
			// eww.yuck's 24 eww-barctl call sites passes --quiet, which
			// suppresses the reply, and an :onclick handler discards the exit
			// code -- so before this, an operational failure here was invisible
			// from the desk. That is how a broken `hyprctl dispatch dpms off`
			// went unnoticed for weeks: the panel simply kept reporting the mode
			// it was already in.
			//
			// Usage errors are deliberately excluded. A bad action name is the
			// caller's mistake, not a condition of the machine, and rewriting
			// the panel to describe it would be noise.
			if !errors.Is(err, collect.ErrDisplayAction) &&
				!errors.Is(err, collect.ErrNoExternalMonitor) {
				failed := collect.DisplayState(err.Error())
				store.Update(func(bar *state.Bar) { bar.Display = failed })
			}
			return nil, err
		}
		idle := collect.IdleInhibitedState()
		store.Update(func(bar *state.Bar) {
			bar.IdleInhibited = idle
			bar.Display = value
		})
		return ok("display", pair("action", action), pair("display", value)), nil

	case "ai":
		action := payloadString(payload, "action", "refresh")
		if action != "refresh" {
			return nil, errValue("ai action must be refresh")
		}
		status := "already-refreshing"
		if QueueAiRefresh(store) {
			status = "queued"
		}
		return ok("ai", pair("action", action), pair("status", status)), nil

	case "notif":
		action := payloadString(payload, "action", "")
		var value collect.NotificationsState
		var err error
		switch action {
		case "toggle-group":
			value, err = collect.ToggleGroup(payloadString(payload, "app", ""))
		case "dismiss":
			value, err = collect.DismissNotification(payloadOr(payload, "id", ""))
		case "clear-group":
			value, err = collect.ClearGroup(payloadString(payload, "app", ""))
		case "clear-all":
			value = collect.ClearAllNotifications()
		case "dnd-toggle":
			value = collect.ToggleDND()
		case "mark-seen":
			value = collect.MarkSeen()
		default:
			err = errValue("notif action must be toggle-group, dismiss, clear-group, " +
				"clear-all, dnd-toggle, or mark-seen")
		}
		if err != nil {
			return nil, err
		}
		store.Update(func(bar *state.Bar) { bar.Notifications = value })
		return ok("notif", pair("action", action),
			pair("notifications", value)), nil

	case "wallpaper":
		action := payloadString(payload, "action", "")
		var value collect.Wallpaper
		var err error
		switch action {
		case "set":
			value, err = collect.SetWallpaper(payloadString(payload, "path", ""))
		case "rescan":
			value = collect.CollectWallpaper()
		default:
			err = errValue("wallpaper action must be set or rescan")
		}
		if err != nil {
			return nil, err
		}
		store.Update(func(bar *state.Bar) { bar.Wallpaper = value })
		return ok("wallpaper", pair("action", action),
			pair("wallpaper", value)), nil
	}
	return nil, errValue("unknown control command")
}

// payloadOr is payload.get(key, fallback) without the string coercion
// payloadString applies: `notif dismiss` accepts anything int() would.
func payloadOr(payload map[string]any, key string, fallback any) any {
	if value, present := payload[key]; present {
		return value
	}
	return fallback
}

type valueError string

func (e valueError) Error() string { return string(e) }

func errValue(message string) error { return valueError(message) }
