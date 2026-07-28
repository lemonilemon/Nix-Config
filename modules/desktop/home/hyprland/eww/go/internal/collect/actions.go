package collect

import (
	"strconv"
	"time"
)

// The system actions the control socket exposes. Each one performs a side
// effect and then re-reads the collector it affected, so the reply carries the
// state the bar should now show rather than the state it asked for.

const actionTimeout = 5 * time.Second

// AdjustVolume mirrors control.adjust_volume.
func AdjustVolume(direction string) (VolumeState, error) {
	var amount string
	switch direction {
	case "up":
		amount = "2%+"
	case "down":
		amount = "2%-"
	default:
		return VolumeState{}, errValue("volume direction must be up or down")
	}
	RunStatus(actionTimeout, "wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", amount)
	// refreshSinks=false: a level nudge cannot add, remove or re-default a
	// sink, and this is the :onscroll path where the two extra pactl forks are
	// the whole cost.
	return CollectVolume(false), nil
}

// SetVolume mirrors control.set_volume.
func SetVolume(value any) (VolumeState, error) {
	number, ok := pyFloatOf(value)
	if !ok {
		return VolumeState{}, errValue("volume set requires a number 0-100")
	}
	// int(round(x)): half-to-even, not away-from-zero, so 0.5 is 0 and 1.5 is 2.
	level := clampInt(RoundHalfEven(number), 0, 100)
	RunStatus(actionTimeout, "wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@",
		strconv.Itoa(level)+"%")
	return CollectVolume(false), nil
}

// pyFloatOf is float(value) for the shapes a control payload carries. A None
// or a non-numeric string raises TypeError/ValueError in the original, both of
// which its handler turns into the same message.
func pyFloatOf(value any) (float64, bool) {
	if text, isText := value.(string); isText {
		return PyFloat(text)
	}
	return pyNumber(value)
}

// ToggleMute mirrors control.toggle_mute.
func ToggleMute() VolumeState {
	RunStatus(actionTimeout, "wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle")
	return CollectVolume(false)
}

// SetSink mirrors control.set_sink.
//
// This one DOES refresh the sink list, unlike the level actions: changing the
// default sink is exactly the event that invalidates the cache.
func SetSink(name string) (VolumeState, error) {
	if name == "" {
		return VolumeState{}, errValue("volume sink requires a sink name")
	}
	RunStatus(actionTimeout, "pactl", "set-default-sink", name)
	return CollectVolume(true), nil
}

// ControlMedia mirrors control.control_media.
func ControlMedia(action string) (MediaState, error) {
	switch action {
	case "play-pause", "next", "previous":
	default:
		return MediaState{}, errValue("media action must be play-pause, next, or previous")
	}
	RunStatus(actionTimeout, "playerctl", action)
	return CollectMedia(), nil
}

// ToggleBluetoothPower mirrors control.toggle_bluetooth_power.
func ToggleBluetoothPower() BluetoothState {
	target := "on"
	if CollectBluetooth().Powered == "true" {
		target = "off"
	}
	RunStatus(actionTimeout, "bluetoothctl", "power", target)
	return CollectBluetooth()
}

// DisconnectBluetooth mirrors control.disconnect_bluetooth.
func DisconnectBluetooth(mac string) (BluetoothState, error) {
	if mac == "" {
		return BluetoothState{}, errValue("bluetooth disconnect requires a device address")
	}
	RunStatus(actionTimeout, "bluetoothctl", "disconnect", mac)
	return CollectBluetooth(), nil
}

// ToggleWifi mirrors control.toggle_wifi.
func ToggleWifi() NetworkStateFull {
	target := "on"
	if NetworkRadioEnabled() == "true" {
		target = "off"
	}
	RunStatus(actionTimeout, "nmcli", "radio", "wifi", target)
	return CollectNetwork()
}
