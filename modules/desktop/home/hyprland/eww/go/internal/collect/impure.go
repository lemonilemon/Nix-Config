package collect

import (
	"strconv"
	"sync"
	"time"
)

// Naming: pure parsers are XFromText / XFromJSON and their result types are
// plain nouns (VolumeState, BluetoothState). The impure wrappers that fork a
// process to feed them are CollectX, so a type and the function producing it
// never collide.

// ActiveWindowState is the shape collectors.active_window_state returns.
type ActiveWindowState struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

// appIcons mirrors collectors.APP_ICONS.
//
// Four entries -- kitty, firefox, spotify, thunar -- are a bare space with no
// glyph, byte-identical to the fallback for an unknown app. Verified against
// the source bytes, not the rendered text. That is almost certainly a past
// encoding accident (this repo has two commits repairing mangled glyphs
// elsewhere), but it is what shipped, so it is what this reproduces. It was left
// alone during the port because fixing it is a behaviour change and had no
// business being smuggled in through a port. That reason has expired -- give
// those four apps real glyphs whenever you like, it is a one-line change here
// and the golden replay will flag it so you know you meant it.
var appIcons = map[string]string{
	"kitty":               " ",
	"zen-beta":            "\U000F0239 ",
	"zen":                 "\U000F0239 ",
	"firefox":             " ",
	"webcord":             "\U000F066F ",
	"spotify":             " ",
	"org.remmina.Remmina": "\U000F08B9 ",
	"thunar":              " ",
	"code":                "\U000F0A1E ",
	"code-url-handler":    "\U000F0A1E ",
}

const appIconFallback = " "

// ActiveWindowStateFrom mirrors collectors.active_window_state's body, taking
// the hyprctl output rather than fetching it.
func ActiveWindowStateFrom(activeWindowJSON string) ActiveWindowState {
	var data map[string]any
	if !ParseJSON(activeWindowJSON, &data) {
		data = map[string]any{}
	}
	class, _ := data["class"].(string)
	title, _ := data["title"].(string)
	if class == "" && title == "" {
		return ActiveWindowState{}
	}
	icon, ok := appIcons[class]
	if !ok {
		icon = appIconFallback
	}
	// Upper bound only -- the bar label has :truncate, so GTK ellipsizes to the
	// actual space available; this just keeps the state JSON bounded.
	display := icon + class
	if title != "" {
		display = icon + class + " · " + TruncateText(title, 60)
	}
	return ActiveWindowState{
		Text:    display,
		Tooltip: class + "\n" + title,
		Class:   class,
	}
}

// ActiveWindowState mirrors collectors.active_window_state.
func CollectActiveWindow() ActiveWindowState {
	return ActiveWindowStateFrom(RunText(defaultTimeout, "hyprctl", "activewindow", "-j"))
}

// WorkspaceState mirrors collectors.workspace_state.
func CollectWorkspace() WorkspaceState {
	return WorkspaceStateFromJSON(
		RunText(defaultTimeout, "hyprctl", "activeworkspace", "-j"),
		RunText(defaultTimeout, "hyprctl", "workspaces", "-j"),
		RunText(defaultTimeout, "hyprctl", "clients", "-j"),
	)
}

// MediaState mirrors collectors.media_state.
func CollectMedia() MediaState {
	return MediaStateFromText(
		RunText(time.Second, "playerctl", "status"),
		RunText(defaultTimeout, "playerctl", "metadata", "--format", "{{artist}} - {{title}}"),
	)
}

// TrayCount mirrors collectors.tray_count.
func CollectTrayCount() int {
	return TrayCountFromText(RunText(defaultTimeout,
		"busctl", "--user", "get-property",
		"org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher",
		"org.kde.StatusNotifierWatcher", "RegisteredStatusNotifierItems",
	))
}

// BluetoothState mirrors collectors.bluetooth_state.
//
// One `bluetoothctl info` per connected device, keyed by address -- so the
// number of children scales with how many devices are paired and connected.
func CollectBluetooth() BluetoothState {
	controller := RunText(defaultTimeout, "bluetoothctl", "show")
	devices := RunText(defaultTimeout, "bluetoothctl", "devices", "Connected")
	infoByAddress := map[string]string{}
	for _, line := range SplitLines(devices) {
		if parts := SplitWhitespaceN(line, 2); len(parts) >= 2 {
			infoByAddress[parts[1]] = RunText(defaultTimeout, "bluetoothctl", "info", parts[1])
		}
	}
	return BluetoothStateFromText(controller, devices, infoByAddress)
}

var (
	sinksLock   sync.Mutex
	cachedSinks []Sink
)

// ResetVolumeSinksCache mirrors collectors.reset_volume_sinks_cache.
func ResetVolumeSinksCache() {
	sinksLock.Lock()
	defer sinksLock.Unlock()
	cachedSinks = nil
}

// VolumeSinks mirrors collectors.volume_sinks. See that docstring: this is
// cached because volume_state runs on every scroll tick and enumerating sinks
// costs two more forks, and only a sink appearing or the default changing can
// alter the list. A cold cache always refreshes.
func VolumeSinks(refresh bool) []Sink {
	if !refresh {
		sinksLock.Lock()
		if cachedSinks != nil {
			out := make([]Sink, len(cachedSinks))
			copy(out, cachedSinks)
			sinksLock.Unlock()
			return out
		}
		sinksLock.Unlock()
	}

	defaultName := Strip(RunText(defaultTimeout, "pactl", "get-default-sink"))
	sinks := SinksFromPactlJSON(RunText(defaultTimeout, "pactl", "-f", "json", "list", "sinks"), defaultName)
	if len(sinks) == 0 {
		sinks = SinksFromPactlShort(RunText(defaultTimeout, "pactl", "list", "short", "sinks"), defaultName)
	}

	sinksLock.Lock()
	cachedSinks = make([]Sink, len(sinks))
	copy(cachedSinks, sinks)
	sinksLock.Unlock()
	return sinks
}

// VolumeState mirrors collectors.volume_state.
func CollectVolume(refreshSinks bool) VolumeState {
	return VolumeStateFromText(
		RunText(defaultTimeout, "wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"),
		VolumeSinks(refreshSinks),
	)
}

// NetworkRadioEnabled mirrors collectors.network_radio_enabled.
func NetworkRadioEnabled() string {
	if Strip(RunText(defaultTimeout, "nmcli", "radio", "wifi")) == "enabled" {
		return "true"
	}
	return "false"
}

// LinkFallbackState mirrors collectors.link_fallback_state.
func CollectLinkFallback() (NetworkState, bool) {
	return LinkStateFromText(
		RunText(defaultTimeout, "ip", "route"),
		RunText(defaultTimeout, "ip", "-o", "-4", "addr", "show"),
	)
}

// NetworkConnectionState mirrors collectors.network_connection_state.
func CollectNetworkConnection() NetworkState {
	status := RunText(defaultTimeout, "nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status")
	wifiDevice := ConnectedDevice(status, "wifi")
	ethernetDevice := ConnectedDevice(status, "ethernet")

	// Whichever interface carries the default route is the one actually in use.
	if ethernetDevice != "" && DefaultRouteDevice(RunText(defaultTimeout, "ip", "route")) == ethernetDevice {
		wifiDevice = ""
	}

	if wifiDevice != "" {
		details := RunText(defaultTimeout,
			"nmcli", "-t", "-f", "GENERAL.CONNECTION,IP4.ADDRESS", "dev", "show", wifiDevice)
		ssid := NmcliValue(details, "GENERAL.CONNECTION")
		if ssid == "" {
			ssid = wifiDevice
		}
		ipInfo := FirstIP(details)
		wirelessText, _ := ReadTextFile("/proc/net/wireless")
		signal, haveSignal := WirelessSignalPercent(wirelessText, wifiDevice)
		if !haveSignal {
			wifi := RunText(defaultTimeout, "nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL", "dev", "wifi")
			return NetworkStateFromText(status, wifi, map[string]string{wifiDevice: details})
		}
		if ipInfo != "" {
			return NetworkState{
				Text:    glyphWifi + " " + ssid + " " + strconv.Itoa(signal) + "%",
				Tooltip: wifiDevice + ": " + ipInfo,
				Class:   "wifi",
			}
		}
		return NetworkState{
			Text:    glyphNoIP + " (No IP)",
			Tooltip: wifiDevice + ": No IP",
			Class:   "linked",
		}
	}

	if ethernetDevice != "" {
		details := RunText(defaultTimeout, "nmcli", "-t", "-f", "IP4.ADDRESS", "dev", "show", ethernetDevice)
		if ipInfo := FirstIP(details); ipInfo != "" {
			return NetworkState{
				Text:    glyphEthernet + " " + ethernetDevice,
				Tooltip: ethernetDevice + ": " + ipInfo,
				Class:   "ethernet",
			}
		}
		return NetworkState{
			Text:    glyphNoIP + " (No IP)",
			Tooltip: ethernetDevice + ": No IP",
			Class:   "linked",
		}
	}

	// nmcli claims nothing is connected. Before believing it, check whether the
	// kernel still has a default route: NetworkManager may be stopped, absent,
	// or simply not the owner of this machine's networking.
	if fallback, ok := CollectLinkFallback(); ok {
		return fallback
	}

	return NetworkState{
		Text:    glyphWarning + " Disconnected",
		Tooltip: "No connection",
		Class:   "disconnected",
	}
}

// NetworkStateFull mirrors collectors.network_state, which is
// network_connection_state plus the radio flag.
type NetworkStateFull struct {
	Text        string `json:"text"`
	Tooltip     string `json:"tooltip"`
	Class       string `json:"class"`
	WifiEnabled string `json:"wifi_enabled"`
}

// NetworkStateFull_ mirrors collectors.network_state.
//
// No EWW_BAR_WIFI counterpart to the battery guard on purpose: a wifi-less host
// hides the toggle in the popup (eww.wifi.enable gates the widget), but the
// radio query is one cheap nmcli call on a path that has to run anyway for the
// connection state, so gating it here would buy nothing.
func CollectNetwork() NetworkStateFull {
	base := CollectNetworkConnection()
	return NetworkStateFull{
		Text:        base.Text,
		Tooltip:     base.Tooltip,
		Class:       base.Class,
		WifiEnabled: NetworkRadioEnabled(),
	}
}
