package collect

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

// Naming: pure parsers are XFromText / XFromJSON with plain-noun result types
// (VolumeState, BluetoothState); the impure wrappers that fork a process to feed
// them are CollectX, so a type and its producer never collide.

type ActiveWindowState struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

// appIcons maps a window class to the glyph the bar prefixes its label with.
//
// Escapes, never literals: every one renders as a blank or a box in an editor, a
// terminal and a diff. The comment beside each is the Nerd Font glyph name, and
// all are Material Design (md-*) so the stroke weights match on one line.
//
// Chosen by rendering at .active-window's real 12px: anything with interior detail
// finer than about two pixels is out, however apt it reads in a picker.
//
// Keys MUST be lowercase and unwrapped -- lookupAppIcon normalises the class before
// indexing, and TestAppIconKeysAreNormalised fails the build otherwise. To add one:
//
//	hyprctl activewindow -j | jq -r .class
//
// An unknown class falls back to a bare space, so a wrong guess costs a missing icon.
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

// normalizeClass folds the spellings of one app's window class together. Case: a
// Wayland app_id is usually lowercase ("code") where the X11 WM_CLASS the same app
// sets under XWayland is capitalised ("Code"). Wrapping: NixOS builds many GTK
// binaries behind a wrapper script, so an app taking its class from argv[0]
// announces itself as ".blueman-manager-wrapped".
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

// ActiveWindowStateFrom takes the hyprctl output rather than fetching it.
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

func CollectActiveWindow() ActiveWindowState {
	return ActiveWindowStateFrom(RunText(defaultTimeout, "hyprctl", "activewindow", "-j"))
}

// CollectWorkspace passes clientsJSON deliberately empty. It feeds the urgency
// branch in WorkspaceStateFromJSON, and Hyprland does not supply the input:
// 0.56.0's `hyprctl clients -j` does not emit `urgent`, so the branch has never
// fired and .workspace.urgent is unreachable CSS.
//
// Passed as "" rather than removed from the signature, which 294 recorded cases in
// the golden replay pin. The win is a fork on every window open, close and move.
// If urgency is wanted back, Hyprland 0.56 exposes it through the Lua API.
func CollectWorkspace() WorkspaceState {
	workspace, _ := CollectWorkspaceViews()
	return workspace
}

// CollectWorkspaceViews reads hyprctl once and derives both workspace models: they
// both need `hyprctl workspaces -j` and refresh on the same events, so separate
// collectors would fork it twice on every window move.
func CollectWorkspaceViews() (WorkspaceState, MonitorWorkspaces) {
	active := RunText(defaultTimeout, "hyprctl", "activeworkspace", "-j")
	workspaces := RunText(defaultTimeout, "hyprctl", "workspaces", "-j")
	monitors := RunText(defaultTimeout, "hyprctl", "monitors", "-j")
	return WorkspaceStateFromJSON(active, workspaces, ""),
		MonitorWorkspacesFromJSON(monitors, workspaces)
}

func CollectMedia() MediaState {
	return MediaStateFromText(
		RunText(time.Second, "playerctl", "status"),
		RunText(defaultTimeout, "playerctl", "metadata", "--format", "{{artist}} - {{title}}"),
	)
}

func CollectTrayCount() int {
	return TrayCountFromText(RunText(defaultTimeout,
		"busctl", "--user", "get-property",
		"org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher",
		"org.kde.StatusNotifierWatcher", "RegisteredStatusNotifierItems",
	))
}

// CollectBluetooth runs one `bluetoothctl info` per connected device, so the number
// of children scales with how many devices are paired and connected.
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

func ResetVolumeSinksCache() {
	sinksLock.Lock()
	defer sinksLock.Unlock()
	cachedSinks = nil
}

// VolumeSinks is cached because volume_state runs on every scroll tick and
// enumerating sinks costs two more forks. A cold cache always refreshes.
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

func CollectVolume(refreshSinks bool) VolumeState {
	return VolumeStateFromText(
		RunText(defaultTimeout, "wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"),
		VolumeSinks(refreshSinks),
	)
}

func NetworkRadioEnabled() string {
	if Strip(RunText(defaultTimeout, "nmcli", "radio", "wifi")) == "enabled" {
		return "true"
	}
	return "false"
}

func CollectLinkFallback() (NetworkState, bool) {
	return LinkStateFromText(
		RunText(defaultTimeout, "ip", "route"),
		RunText(defaultTimeout, "ip", "-o", "-4", "addr", "show"),
	)
}

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
	// kernel still has a default route: NetworkManager may be stopped or absent.
	if fallback, ok := CollectLinkFallback(); ok {
		return fallback
	}

	return NetworkState{
		Text:    glyphWarning + " Disconnected",
		Tooltip: "No connection",
		Class:   "disconnected",
	}
}

// CollectNetworkIdentity reads which connection is carrying traffic. Two cheap
// forks on the timer and watcher the rest of the network state already uses.
func CollectNetworkIdentity() NetworkIdentity {
	return ActiveConnectionFromText(
		RunText(defaultTimeout, "nmcli", "-t", "-f", "UUID,NAME,TYPE,DEVICE",
			"connection", "show", "--active"),
		RunText(defaultTimeout, "ip", "route"),
	)
}

// CollectConnectivity reads NetworkManager's cached verdict on whether this link
// reaches the internet, NOT `connectivity check`: NetworkManager re-runs its own
// probe when a link comes up, so the cached answer converges within a second or
// two, where forcing a check would put a network round trip on a path that must
// never block.
func CollectConnectivity() string {
	return ConnectivityFromText(
		RunText(defaultTimeout, "nmcli", "-t", "networking", "connectivity"))
}

// NetworkStateFull is the connection state plus the radio flag, the connection
// identity, NetworkManager's connectivity verdict, and the speed card.
type NetworkStateFull struct {
	Text         string    `json:"text"`
	Tooltip      string    `json:"tooltip"`
	Class        string    `json:"class"`
	WifiEnabled  string    `json:"wifi_enabled"`
	Connectivity string    `json:"connectivity"`
	ConnUUID     string    `json:"conn_uuid"`
	ConnName     string    `json:"conn_name"`
	Speed        Speedtest `json:"speed"`
	Policy       NetPolicy `json:"policy"`
}

// CollectNetwork has no EWW_BAR_WIFI counterpart to the battery guard on purpose:
// the radio query is one cheap nmcli call on a path that runs anyway.
func CollectNetwork() NetworkStateFull {
	base := CollectNetworkConnection()
	identity := CollectNetworkIdentity()
	connectivity := CollectConnectivity()
	return NetworkStateFull{
		Text:         base.Text,
		Tooltip:      base.Tooltip,
		Class:        base.Class,
		WifiEnabled:  NetworkRadioEnabled(),
		Connectivity: connectivity,
		ConnUUID:     identity.UUID,
		ConnName:     identity.Name,
		// Assembled from the record cache rather than collected, so every caller
		// lands a card consistent with the identity it just read, with no extra
		// fork and no second code path.
		Speed: SpeedtestCardFor(identity.UUID, base.Class, connectivity, 0),
		// Assembled from its cache the same way, and for the same reason.
		Policy: NetPolicyCardFor(identity.UUID, base.Class, 0),
	}
}
