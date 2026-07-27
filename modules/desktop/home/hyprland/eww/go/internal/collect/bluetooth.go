package collect

import (
	"fmt"
	"regexp"
	"strings"
)

const glyphBluetoothConnected = "\U000F00AF" // U+F00AF bluetooth

var (
	batteryRe   = regexp.MustCompile(`Battery Percentage:.*\((\d+)\)`)
	preferredRe = regexp.MustCompile(`(?i)ugreen_1|ugreen_2`)
)

// BluetoothDevice is one entry of the bluetooth popup's device list.
type BluetoothDevice struct {
	Mac       string `json:"mac"`
	Name      string `json:"name"`
	Battery   string `json:"battery"`
	Connected string `json:"connected"`
}

// BluetoothState is the shape collectors.bluetooth_state_from_text returns.
type BluetoothState struct {
	Text    string            `json:"text"`
	Tooltip string            `json:"tooltip"`
	Class   string            `json:"class"`
	Powered string            `json:"powered"`
	Devices []BluetoothDevice `json:"devices"`
}

// ParseController mirrors collectors.parse_controller.
//
// The bool is the point. bluetoothctl prints "Powered: yes", and passing that
// raw string outward is what made eww.scss's .bluetooth.off rule unreachable
// for as long as it existed -- the class rendered as "no". Normalising at the
// parse boundary is the fix, so the return type here is deliberately not a
// string.
func ParseController(controllerText string) (alias, address string, powered bool) {
	for _, line := range SplitLines(controllerText) {
		stripped := Strip(line)
		switch {
		case strings.HasPrefix(stripped, "Controller "):
			if parts := SplitWhitespaceN(stripped, -1); len(parts) >= 2 {
				address = parts[1]
			}
		case strings.HasPrefix(stripped, "Alias:"):
			_, rest, _ := strings.Cut(stripped, ":")
			alias = Strip(rest)
		case strings.HasPrefix(stripped, "Powered:"):
			_, rest, _ := strings.Cut(stripped, ":")
			powered = Strip(rest) == "yes"
		}
	}
	if alias == "" {
		alias = "Bluetooth"
	}
	if address == "" {
		address = "N/A"
	}
	return alias, address, powered
}

// ParseDeviceInfo mirrors collectors.parse_device_info.
func ParseDeviceInfo(infoText, fallbackAlias string) (alias, battery string) {
	alias = fallbackAlias
	for _, line := range SplitLines(infoText) {
		stripped := Strip(line)
		if strings.HasPrefix(stripped, "Alias:") {
			_, rest, _ := strings.Cut(stripped, ":")
			alias = Strip(rest)
		}
		if match := batteryRe.FindStringSubmatch(stripped); match != nil {
			battery = match[1] + "%"
		}
	}
	return alias, battery
}

// BluetoothStateFromText mirrors collectors.bluetooth_state_from_text.
func BluetoothStateFromText(controllerText, devicesText string, infoByAddress map[string]string) BluetoothState {
	controllerAlias, controllerAddress, powered := ParseController(controllerText)
	poweredFlag := "false"
	if powered {
		poweredFlag = "true"
	}

	var devices []string
	for _, line := range SplitLines(devicesText) {
		if Strip(line) != "" {
			devices = append(devices, line)
		}
	}

	structured := []BluetoothDevice{}
	for _, line := range devices {
		parts := SplitWhitespaceN(line, 2)
		if len(parts) < 2 {
			continue
		}
		address := parts[1]
		fallback := address
		if len(parts) >= 3 {
			fallback = parts[2]
		}
		alias, battery := ParseDeviceInfo(infoByAddress[address], fallback)
		structured = append(structured, BluetoothDevice{
			Mac:       address,
			Name:      TruncateText(alias, 24),
			Battery:   battery,
			Connected: "true",
		})
	}

	if len(devices) == 0 {
		class := ""
		if !powered {
			class = "off"
		}
		return BluetoothState{
			Text:    "",
			Tooltip: fmt.Sprintf("%s\t%s\n\n0 connected", controllerAlias, controllerAddress),
			Class:   class,
			Powered: poweredFlag,
			Devices: []BluetoothDevice{},
		}
	}

	preferred := devices[0]
	for _, line := range devices {
		if preferredRe.MatchString(line) {
			preferred = line
			break
		}
	}
	preferredParts := SplitWhitespaceN(preferred, 2)
	firstAddress := ""
	if len(preferredParts) >= 2 {
		firstAddress = preferredParts[1]
	}
	fallbackAlias := firstAddress
	if len(preferredParts) >= 3 {
		fallbackAlias = preferredParts[2]
	}
	firstAlias, firstBattery := ParseDeviceInfo(infoByAddress[firstAddress], fallbackAlias)

	text := fmt.Sprintf("%s %s", glyphBluetoothConnected, firstAlias)
	if firstBattery != "" {
		text = fmt.Sprintf("%s %s %s", glyphBluetoothConnected, firstAlias, firstBattery)
	}
	text = TruncateText(text, 24)

	detail := make([]string, 0, len(structured))
	for _, device := range structured {
		if device.Battery != "" {
			detail = append(detail, fmt.Sprintf("%s\t%s\t%s", device.Name, device.Mac, device.Battery))
		} else {
			detail = append(detail, fmt.Sprintf("%s\t%s", device.Name, device.Mac))
		}
	}

	return BluetoothState{
		Text: text,
		Tooltip: fmt.Sprintf(
			"%s\t%s\n\n%d connected\n\n%s",
			controllerAlias, controllerAddress, len(devices), strings.Join(detail, "\n"),
		),
		Class:   "connected",
		Powered: poweredFlag,
		Devices: structured,
	}
}
