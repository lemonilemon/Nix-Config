package collect

import (
	"strconv"
	"strings"
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

// appIcons maps a window class to the glyph the bar prefixes its label with.
//
// Escapes, never literals. Every one of these renders as a blank or a box in an
// editor, a terminal and a diff, so a literal cannot be reviewed and does not
// survive a copy-paste. The comment beside each is the Nerd Font glyph name, and
// all of them are Material Design (md-*) on purpose: mixing in fa-*, dev-* or
// cod-* gives visibly different stroke weights and optical sizes on one line.
//
// Brand glyph where the font has one, function otherwise. That is why kitty is a
// cat and zathura is a document.
//
// Chosen by rendering at .active-window's real 12px, not by name. Three obvious
// picks failed there: md-api and md-file_pdf_box draw the letters "API" and
// "PDF" inside a box, which is a smudge at that size, and md-notebook's spiral
// binding fills in solid. Anything with interior detail finer than about two
// pixels is out, however apt it reads in a picker.
//
// Keys MUST be lowercase and unwrapped -- lookupAppIcon normalises the class
// before indexing, and TestAppIconKeysAreNormalised fails the build otherwise.
//
// To add one, focus the window and read its class off the daemon:
//
//	hyprctl activewindow -j | jq -r .class
//
// An unknown class is not an error; it falls back to a bare space, so a wrong
// guess here costs nothing but a missing icon.
var appIcons = map[string]string{
	// terminals
	"kitty": "\U000F011B ", // md-cat

	// browsers
	"zen-beta": "\U000F0239 ", // md-firefox
	"zen":      "\U000F0239 ", // md-firefox
	"firefox":  "\U000F0239 ", // md-firefox

	// chat
	"webcord": "\U000F066F ", // md-discord

	// media
	"spotify": "\U000F04C7 ", // md-spotify
	"vlc":     "\U000F057C ", // md-vlc

	// files
	"nemo":   "\U000F0770 ", // md-folder_open
	"thunar": "\U000F0770 ", // md-folder_open

	// editors and dev tools
	"code":                    "\U000F0A1E ", // md-microsoft_visual_studio_code
	"code-url-handler":        "\U000F0A1E ", // md-microsoft_visual_studio_code
	"ai.opencode.desktop":     "\U000F06A9 ", // md-robot
	"postman":                 "\U000F048A ", // md-send
	"podman desktop":          "\U000F01A7 ", // md-cube_outline
	"org.wireshark.wireshark": "\U000F1673 ", // md-shark_fin
	"org.remmina.remmina":     "\U000F08B9 ", // md-remote_desktop

	// reading and writing
	"org.pwmt.zathura": "\U000F09EE ", // md-file_document_outline
	"obsidian":         "\U000F104A ", // md-graph_outline
	"imv":              "\U000F02E9 ", // md-image

	// utilities
	"1password":       "\U000F0BC4 ", // md-shield_key
	"gopeed":          "\U000F01DA ", // md-download
	"nwg-displays":    "\U000F0379 ", // md-monitor
	"blueman-manager": "\U000F00AF ", // md-bluetooth
	"tradingview":     "\U000F012A ", // md-chart_line
}

const appIconFallback = " "

// normalizeClass folds the spellings of one app's window class together.
//
// Two things vary and neither is the app's fault. Case: a Wayland app_id is
// usually lowercase ("code") while the X11 WM_CLASS the same app sets under
// XWayland is capitalised ("Code"), and hyprctl reports whichever it got.
// Wrapping: NixOS builds many GTK binaries behind a wrapper script, and an app
// that takes its class from argv[0] then announces itself as
// ".blueman-manager-wrapped".
//
// Folding here rather than adding a key per spelling keeps appIcons readable and
// means a new app needs one entry, not four.
func normalizeClass(class string) string {
	folded := strings.ToLower(class)
	folded = strings.TrimPrefix(folded, ".")
	return strings.TrimSuffix(folded, "-wrapped")
}

// lookupAppIcon returns the glyph for a window class, or the fallback.
func lookupAppIcon(class string) string {
	if icon, ok := appIcons[normalizeClass(class)]; ok {
		return icon
	}
	return appIconFallback
}

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
	icon := lookupAppIcon(class)
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
//
// clientsJSON is deliberately empty. It exists to feed the urgency branch in
// WorkspaceStateFromJSON, and Hyprland does not supply the input: 0.56.0's
// `hyprctl clients -j` emits 32 keys per window and `urgent` is not among them,
// so the branch has never fired on this machine and .workspace.urgent is
// unreachable CSS.
//
// Passing "" rather than deleting the parameter, because WorkspaceStateFromJSON
// is pinned by 294 recorded cases in the golden replay, which cannot be
// regenerated. Its signature and body stay exactly as recorded; only what
// production chooses to hand it changes, and the recorded cases replay with
// their own arguments so they never see this.
//
// The win is a fork. This runs on every workspace event -- every window open,
// close and move, and every workspace switch -- and `hyprctl clients -j` is the
// most expensive of the three calls, since it serialises every window on the
// system. If urgency is ever wanted back, Hyprland 0.56 exposes it through the
// Lua API as window.urgent rather than through the JSON.
func CollectWorkspace() WorkspaceState {
	return WorkspaceStateFromJSON(
		RunText(defaultTimeout, "hyprctl", "activeworkspace", "-j"),
		RunText(defaultTimeout, "hyprctl", "workspaces", "-j"),
		"",
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
