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

// ConnectedDevice mirrors collectors.connected_device.
func ConnectedDevice(statusText, deviceType string) string {
	for _, line := range SplitLines(statusText) {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[1] == deviceType && fields[2] == "connected" {
			return fields[0]
		}
	}
	return ""
}

// NmcliValue mirrors collectors.nmcli_value.
func NmcliValue(text, key string) string {
	for _, line := range SplitLines(text) {
		if strings.HasPrefix(line, key+":") {
			_, rest, _ := strings.Cut(line, ":")
			return rest
		}
	}
	return ""
}

// FirstIP mirrors collectors.first_ip.
func FirstIP(ipText string) string {
	for _, line := range SplitLines(ipText) {
		if strings.Contains(line, ":") {
			_, rest, _ := strings.Cut(line, ":")
			return rest
		}
	}
	return ""
}

// WirelessSignalPercent mirrors collectors.wireless_signal_percent.
//
// ok=false is Python's None, and the caller treats it as "fall back to asking
// nmcli", not as "signal is zero". /proc/net/wireless reports link quality out
// of 70, not a percentage.
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

// DefaultRouteDevice mirrors collectors.default_route_device.
//
// Interface owning the default route, lowest metric winning. A docked laptop
// carries one default route per link, and the kernel prefers the lowest metric;
// an absent metric means 0. ECMP routes (a bare `default` line followed by
// `nexthop` lines) are deliberately not handled - they have no single owning
// interface to name.
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

// NetworkIdentity names the connection carrying traffic, as NetworkManager
// knows it.
//
// UUID is the key every speed test record is filed under, and it is the right
// one for the job where the obvious alternatives are not. An SSID is not
// unique -- every branch of a chain is "Starbucks", and keying on it would show
// one cafe's numbers in another. A BSSID is too granular: a hotel with twenty
// access points would accumulate twenty records and roaming would invalidate
// them constantly. A gateway MAC works but is a heuristic to maintain, and
// NetworkManager already computes a stable per-profile identity.
//
// The one case UUID does not separate: two genuinely different networks that
// share an SSID also share one NetworkManager profile, so they share one
// record. Accepted -- splitting them needs a gateway-MAC sub-key and buys
// almost nothing.
type NetworkIdentity struct {
	UUID string
	Name string
}

// splitNmcliTerse splits one `nmcli -t` line into its fields.
//
// Not strings.Split(line, ":"), which is what the older collectors above use.
// Terse mode escapes a colon inside a VALUE as `\:` and a backslash as `\\`,
// and a connection name is free text -- an SSID of "Guest: Lobby" would
// otherwise split into two fields and shift every column after it, so the UUID
// would be read out of the wrong position.
//
// NetworkStateFromText above has the same exposure and does NOT use this: it
// splits an ACTIVE:SSID:SIGNAL line on bare colons, so an SSID containing one
// reads as a truncated name and a signal of nonsense. That is a live bug, left
// alone deliberately -- that function is pinned byte-for-byte by recorded cases
// in the golden replay, which cannot be regenerated, so correcting it needs the
// supersession treatment rather than an edit in passing.
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

// ActiveConnectionFromText picks the identity out of
// `nmcli -t -f UUID,NAME,TYPE,DEVICE connection show --active`.
//
// The connection owning the default route wins, because that is the one
// actually carrying traffic -- a docked laptop, or a machine with a VPN up,
// has several active at once. Falling back to the first non-loopback line
// rather than to nothing keeps the card working when the route text is
// unavailable, which is the same trade DefaultRouteDevice's callers make.
//
// Loopback is always skipped: it is active on every machine and would otherwise
// win the fallback on a disconnected one, filing records under an identity that
// says nothing about the network.
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

// ConnectivityFromText maps `nmcli networking connectivity` onto the popup's
// vocabulary.
//
// The four words are NetworkManager's own and are passed through unchanged;
// anything else, including the empty string run_text() produces when nmcli is
// missing or times out, becomes "unknown". That distinction matters at the
// widget: "none" is a claim that there is no internet, "unknown" is an
// admission that nothing was asked, and the popup must not render the second
// as the first.
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

// DeviceIPv4 mirrors collectors.device_ipv4.
//
// `ip -o -4 addr show` lines look like:
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

// LinkStateFromText mirrors collectors.link_state_from_text.
//
// Readout for when nmcli reports nothing usable. ok=false is Python's None, and
// the distinction carries the whole workaround: None means "nmcli is right,
// we really are disconnected", a value means "nmcli is not reporting but the
// kernel still has a route, so we are online".
//
// The kernel keeps the lease and the default route after NetworkManager stops,
// so the machine is still online; only the usual source of truth is gone. The
// same holds when NetworkManager is up but owns nothing, e.g. unmanaged devices
// or a route belonging to systemd-networkd or a tunnel. Report the interface
// that owns the route rather than claiming to be disconnected.
//
// run_text() flattens "not running", "not on PATH" and "timed out" into the
// same empty string, so the tooltip names where the answer came from instead of
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

// NetworkStateFromText mirrors collectors.network_state_from_text.
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
