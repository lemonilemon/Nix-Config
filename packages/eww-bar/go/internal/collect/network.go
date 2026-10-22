package collect

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	glyphWifi     = "\uF1EB"     // U+F1EB wifi
	glyphNoIP     = "\U000F0200" // U+F0200 ethernet port -- linked but no address
	glyphEthernet = "\U000F0317" // U+F0317 lan
	glyphWarning  = "\u26A0"     // U+26A0 warning sign
)

// NetworkState is the shape the network collectors return, before
// network_state() adds wifi_enabled.
type NetworkState struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

func ConnectedDevice(statusText, deviceType string) string {
	for _, line := range SplitLines(statusText) {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[1] == deviceType && fields[2] == "connected" {
			return fields[0]
		}
	}
	return ""
}

func NmcliValue(text, key string) string {
	for _, line := range SplitLines(text) {
		if strings.HasPrefix(line, key+":") {
			_, rest, _ := strings.Cut(line, ":")
			return rest
		}
	}
	return ""
}

func FirstIP(ipText string) string {
	for _, line := range SplitLines(ipText) {
		if strings.Contains(line, ":") {
			_, rest, _ := strings.Cut(line, ":")
			return rest
		}
	}
	return ""
}

// WirelessSignalPercent reports ok=false as "fall back to asking nmcli", not as
// "signal is zero". /proc/net/wireless reports link quality out of 70.
func WirelessSignalPercent(wirelessText, iface string) (int, bool) {
	for _, line := range SplitLines(wirelessText) {
		stripped := Strip(line)
		if !strings.HasPrefix(stripped, iface+":") {
			continue
		}
		parts := SplitWhitespaceN(strings.ReplaceAll(stripped, ":", " "), -1)
		if len(parts) < 3 {
			return 0, false
		}
		quality, err := strconv.ParseFloat(strings.TrimRight(parts[2], "."), 64)
		if err != nil {
			return 0, false
		}
		return clampInt(RoundHalfEven(quality*100/70), 0, 100), true
	}
	return 0, false
}

// DefaultRouteDevice names the interface owning the default route, lowest metric
// winning; an absent metric means 0. ECMP routes (a bare `default` followed by
// `nexthop` lines) are deliberately not handled: no single owning interface.
func DefaultRouteDevice(routeText string) string {
	bestDevice := ""
	bestMetric := 0
	haveBest := false
	for _, line := range SplitLines(routeText) {
		fields := SplitWhitespaceN(line, -1)
		if len(fields) == 0 || fields[0] != "default" {
			continue
		}
		deviceIndex := indexOf(fields, "dev")
		if deviceIndex < 0 || deviceIndex+1 >= len(fields) {
			continue
		}
		metric := 0
		if metricIndex := indexOf(fields, "metric"); metricIndex >= 0 && metricIndex+1 < len(fields) {
			if parsed, err := strconv.Atoi(fields[metricIndex+1]); err == nil {
				metric = parsed
			}
		}
		if !haveBest || metric < bestMetric {
			bestDevice = fields[deviceIndex+1]
			bestMetric = metric
			haveBest = true
		}
	}
	return bestDevice
}

// NetworkIdentity names the connection carrying traffic, as NetworkManager knows it.
//
// UUID is the key every speed test record is filed under. An SSID is not unique --
// every branch of a chain is "Starbucks" -- and a BSSID is too granular, since a
// hotel with twenty access points would accumulate twenty records.
//
// The one case UUID does not separate: two different networks sharing an SSID also
// share one NetworkManager profile, so they share one record. Accepted.
type NetworkIdentity struct {
	UUID string
	Name string
}

// splitNmcliTerse splits one `nmcli -t` line into its fields. Not
// strings.Split(line, ":"): terse mode escapes a colon inside a VALUE as `\:`, and
// an SSID of "Guest: Lobby" would otherwise shift every column after it.
//
// NetworkStateFromText has the same exposure and does NOT use this, so an SSID
// containing a colon reads as a truncated name. A live bug, left alone: that
// function is pinned byte-for-byte by recorded cases that cannot be regenerated, so
// correcting it needs the supersession treatment rather than an edit in passing.
func splitNmcliTerse(line string) []string {
	fields := []string{}
	current := strings.Builder{}
	escaped := false
	for _, char := range line {
		switch {
		case escaped:
			current.WriteRune(char)
			escaped = false
		case char == '\\':
			escaped = true
		case char == ':':
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}
	return append(fields, current.String())
}

// ActiveConnectionFromText picks the identity out of `nmcli -t -f UUID,NAME,TYPE,
// DEVICE connection show --active`. The connection owning the default route wins,
// because a docked laptop or a machine with a VPN up has several active at once.
//
// Loopback is always skipped: it is active on every machine and would otherwise win
// the fallback on a disconnected one.
func ActiveConnectionFromText(activeText, routeText string) NetworkIdentity {
	preferred := DefaultRouteDevice(routeText)
	fallback := NetworkIdentity{}

	for _, line := range SplitLines(activeText) {
		fields := splitNmcliTerse(line)
		if len(fields) < 4 {
			continue
		}
		uuid, name, kind, device := fields[0], fields[1], fields[2], fields[3]
		if uuid == "" || kind == "loopback" {
			continue
		}
		if preferred != "" && device == preferred {
			return NetworkIdentity{UUID: uuid, Name: name}
		}
		if fallback.UUID == "" {
			fallback = NetworkIdentity{UUID: uuid, Name: name}
		}
	}
	return fallback
}

// ConnectivityFromText passes NetworkManager's own four words through unchanged;
// anything else, including the empty string run_text() produces when nmcli is
// missing, becomes "unknown". The distinction matters at the widget: "none" claims
// there is no internet, "unknown" admits nothing was asked.
func ConnectivityFromText(text string) string {
	switch state := Strip(text); state {
	case "full", "portal", "limited", "none":
		return state
	}
	return "unknown"
}

func indexOf(fields []string, want string) int {
	for i, field := range fields {
		if field == want {
			return i
		}
	}
	return -1
}

// DeviceIPv4 reads `ip -o -4 addr show` lines, which look like:
//
//	2: eno1    inet 192.168.0.88/24 brd ... scope global dynamic eno1
func DeviceIPv4(addrText, device string) string {
	for _, line := range SplitLines(addrText) {
		fields := SplitWhitespaceN(line, -1)
		if len(fields) >= 4 && fields[1] == device && fields[2] == "inet" {
			address, _, _ := strings.Cut(fields[3], "/")
			return address
		}
	}
	return ""
}

// LinkStateFromText is the readout for when nmcli reports nothing usable. ok=false
// means "nmcli is right, we really are disconnected"; a value means "nmcli is not
// reporting but the kernel still has a route, so we are online".
//
// The kernel keeps the lease and the default route after NetworkManager stops, and
// the same holds for unmanaged devices or a route owned by systemd-networkd.
//
// run_text() flattens "not running", "not on PATH" and "timed out" into the same
// empty string, so the tooltip names where the answer came from rather than
// asserting a cause it cannot know.
func LinkStateFromText(routeText, addrText string) (NetworkState, bool) {
	device := DefaultRouteDevice(routeText)
	if device == "" {
		return NetworkState{}, false
	}
	ipInfo := DeviceIPv4(addrText, device)
	if ipInfo == "" {
		ipInfo = "No IP"
	}
	return NetworkState{
		Text: fmt.Sprintf("%s %s %s", glyphEthernet, device, glyphWarning),
		Tooltip: fmt.Sprintf(
			"%s: %s (kernel routing table; NetworkManager not reporting)", device, ipInfo,
		),
		Class: "degraded",
	}, true
}

func NetworkStateFromText(statusText, wifiText string, ipByDevice map[string]string) NetworkState {
	wifiDevice := ""
	for _, line := range SplitLines(statusText) {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[1] == "wifi" && fields[2] == "connected" {
			wifiDevice = fields[0]
			break
		}
	}

	wifi := ""
	for _, line := range SplitLines(wifiText) {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[0] == "yes" {
			wifi = line
			break
		}
	}

	if wifiDevice != "" && wifi != "" {
		fields := strings.Split(wifi, ":")
		ssid, signal := fields[1], fields[2]
		if ipInfo := FirstIP(ipByDevice[wifiDevice]); ipInfo != "" {
			return NetworkState{
				Text:    fmt.Sprintf("%s %s %s%%", glyphWifi, ssid, signal),
				Tooltip: fmt.Sprintf("%s: %s", wifiDevice, ipInfo),
				Class:   "wifi",
			}
		}
		return NetworkState{
			Text:    glyphNoIP + " (No IP)",
			Tooltip: wifiDevice + ": No IP",
			Class:   "linked",
		}
	}

	for _, line := range SplitLines(statusText) {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[1] == "ethernet" && fields[2] == "connected" {
			device := fields[0]
			if ipInfo := FirstIP(ipByDevice[device]); ipInfo != "" {
				return NetworkState{
					Text:    fmt.Sprintf("%s %s", glyphEthernet, device),
					Tooltip: fmt.Sprintf("%s: %s", device, ipInfo),
					Class:   "ethernet",
				}
			}
			return NetworkState{
				Text:    glyphNoIP + " (No IP)",
				Tooltip: device + ": No IP",
				Class:   "linked",
			}
		}
	}

	return NetworkState{
		Text:    glyphWarning + " Disconnected",
		Tooltip: "No connection",
		Class:   "disconnected",
	}
}
