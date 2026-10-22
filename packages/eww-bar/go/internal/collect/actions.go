package collect

import (
	"strconv"
	"time"
)

// The system actions the control socket exposes. Each one performs a side
// effect and then re-reads the collector it affected, so the reply carries the
// state the bar should now show rather than the state it asked for.

const actionTimeout = 5 * time.Second

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
	// refreshSinks=false: a level nudge cannot add, remove or re-default a sink,
	// and this is the :onscroll path where the two extra pactl forks are the cost.
	return CollectVolume(false), nil
}

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

// pyFloatOf is float(value) for the shapes a control payload carries. A None or a
// non-numeric string raises in the original, both turned into the same message.
func pyFloatOf(value any) (float64, bool) {
	if text, isText := value.(string); isText {
		return PyFloat(text)
	}
	return pyNumber(value)
}

func ToggleMute() VolumeState {
	RunStatus(actionTimeout, "wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle")
	return CollectVolume(false)
}

// This one DOES refresh the sink list, unlike the level actions: changing the
// default sink is exactly the event that invalidates the cache.
func SetSink(name string) (VolumeState, error) {
	if name == "" {
		return VolumeState{}, errValue("volume sink requires a sink name")
	}
	RunStatus(actionTimeout, "pactl", "set-default-sink", name)
	return CollectVolume(true), nil
}

func ControlMedia(action string) (MediaState, error) {
	switch action {
	case "play-pause", "next", "previous":
	default:
		return MediaState{}, errValue("media action must be play-pause, next, or previous")
	}
	RunStatus(actionTimeout, "playerctl", action)
	return CollectMedia(), nil
}

func ToggleBluetoothPower() BluetoothState {
	target := "on"
	if CollectBluetooth().Powered == "true" {
		target = "off"
	}
	RunStatus(actionTimeout, "bluetoothctl", "power", target)
	return CollectBluetooth()
}

func DisconnectBluetooth(mac string) (BluetoothState, error) {
	if mac == "" {
		return BluetoothState{}, errValue("bluetooth disconnect requires a device address")
	}
	RunStatus(actionTimeout, "bluetoothctl", "disconnect", mac)
	return CollectBluetooth(), nil
}

func ToggleWifi() NetworkStateFull {
	target := "on"
	if NetworkRadioEnabled() == "true" {
		target = "off"
	}
	RunStatus(actionTimeout, "nmcli", "radio", "wifi", target)
	return CollectNetwork()
}
