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
