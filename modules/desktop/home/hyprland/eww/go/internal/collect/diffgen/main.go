// Command diffgen answers one JSON-encoded call per input line.
//
// It exists so the ported collectors can be checked against the Python they
// came from on the same inputs: tests/eww_bar_backend/test_go_equivalence.py
// feeds both sides the same case file and diffs the answers. Table tests pin
// the cases someone thought of; this pins the ones nobody did.
//
// Protocol, one JSON object per line in, one per line out:
//
//	{"fn": "TruncateText", "args": ["hello", 4]}  ->  {"ok": true, "value": "h..."}
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"ewwbar/internal/collect"
)

type call struct {
	Fn   string            `json:"fn"`
	Args []json.RawMessage `json:"args"`
}

func main() {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 1<<20)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for in.Scan() {
		line := in.Bytes()
		if len(line) == 0 {
			continue
		}
		var c call
		if err := json.Unmarshal(line, &c); err != nil {
			emit(out, nil, err)
			continue
		}
		value, err := dispatch(c)
		emit(out, value, err)
	}
}

func emit(out *bufio.Writer, value any, err error) {
	result := map[string]any{"ok": err == nil}
	if err != nil {
		result["error"] = err.Error()
	} else {
		result["value"] = value
	}
	encoded, _ := json.Marshal(result)
	out.Write(encoded)
	out.WriteByte('\n')
}

func dispatch(c call) (any, error) {
	switch c.Fn {
	case "TruncateText":
		var text string
		var maxLen int
		if err := args(c, &text, &maxLen); err != nil {
			return nil, err
		}
		return collect.TruncateText(text, maxLen), nil

	case "RoundHalfEven":
		var value float64
		if err := args(c, &value); err != nil {
			return nil, err
		}
		return collect.RoundHalfEven(value), nil

	case "PercentPart":
		var value, total float64
		if err := args(c, &value, &total); err != nil {
			return nil, err
		}
		return collect.PercentPart(value, total), nil

	case "ClampPercent":
		var value float64
		if err := args(c, &value); err != nil {
			return nil, err
		}
		return collect.ClampPercent(value), nil

	case "FormatTokens":
		var value float64
		if err := args(c, &value); err != nil {
			return nil, err
		}
		return collect.FormatTokens(value), nil

	case "FormatCost":
		var value float64
		if err := args(c, &value); err != nil {
			return nil, err
		}
		return collect.FormatCost(value), nil

	case "FormatRemaining":
		var seconds float64
		if err := args(c, &seconds); err != nil {
			return nil, err
		}
		return collect.FormatRemaining(seconds), nil

	case "VolumeLabelFromText":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.VolumeLabelFromText(text), nil

	case "VolumeStateFromText":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.VolumeStateFromText(text, nil), nil

	case "SplitLines":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		lines := collect.SplitLines(text)
		if lines == nil {
			lines = []string{}
		}
		return lines, nil

	case "SplitWhitespaceN":
		var text string
		var n int
		if err := args(c, &text, &n); err != nil {
			return nil, err
		}
		fields := collect.SplitWhitespaceN(text, n)
		if fields == nil {
			fields = []string{}
		}
		return fields, nil

	case "Strip":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.Strip(text), nil

	case "ParseController":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		alias, address, powered := collect.ParseController(text)
		return []any{alias, address, powered}, nil

	case "ParseDeviceInfo":
		var infoText, fallback string
		if err := args(c, &infoText, &fallback); err != nil {
			return nil, err
		}
		alias, battery := collect.ParseDeviceInfo(infoText, fallback)
		return []any{alias, battery}, nil

	case "BluetoothStateFromText":
		var controllerText, devicesText string
		var info map[string]string
		if err := args(c, &controllerText, &devicesText, &info); err != nil {
			return nil, err
		}
		return collect.BluetoothStateFromText(controllerText, devicesText, info), nil

	case "SubmapFromEvent":
		var line string
		if err := args(c, &line); err != nil {
			return nil, err
		}
		value, ok := collect.SubmapFromEvent(line)
		if !ok {
			return nil, nil // Python's None
		}
		return value, nil

	case "TrayCountFromText":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.TrayCountFromText(text), nil

	case "MemoryStateFromText":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.MemoryStateFromText(text), nil

	case "MediaStateFromText":
		var statusText, metadataText string
		if err := args(c, &statusText, &metadataText); err != nil {
			return nil, err
		}
		return collect.MediaStateFromText(statusText, metadataText), nil

	case "MonitorEvent":
		var line string
		if err := args(c, &line); err != nil {
			return nil, err
		}
		action, name, ok := collect.MonitorEvent(line)
		if !ok {
			return nil, nil // Python's None
		}
		return []any{action, name}, nil

	case "BarWindowCommand":
		var action, name string
		if err := args(c, &action, &name); err != nil {
			return nil, err
		}
		return collect.BarWindowCommand(action, name), nil

	case "OpenBarNames":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		names := []string{}
		for name := range collect.OpenBarNames(text) {
			names = append(names, name)
		}
		sort.Strings(names) // the Python side is a set; compare sorted
		return names, nil

	case "ConnectedDevice":
		var statusText, deviceType string
		if err := args(c, &statusText, &deviceType); err != nil {
			return nil, err
		}
		return collect.ConnectedDevice(statusText, deviceType), nil

	case "NmcliValue":
		var text, key string
		if err := args(c, &text, &key); err != nil {
			return nil, err
		}
		return collect.NmcliValue(text, key), nil

	case "FirstIP":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.FirstIP(text), nil

	case "WirelessSignalPercent":
		var text, iface string
		if err := args(c, &text, &iface); err != nil {
			return nil, err
		}
		value, ok := collect.WirelessSignalPercent(text, iface)
		if !ok {
			return nil, nil // Python's None
		}
		return value, nil

	case "DefaultRouteDevice":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.DefaultRouteDevice(text), nil

	case "DeviceIPv4":
		var addrText, device string
		if err := args(c, &addrText, &device); err != nil {
			return nil, err
		}
		return collect.DeviceIPv4(addrText, device), nil

	case "LinkStateFromText":
		var routeText, addrText string
		if err := args(c, &routeText, &addrText); err != nil {
			return nil, err
		}
		state, ok := collect.LinkStateFromText(routeText, addrText)
		if !ok {
			return nil, nil // Python's None
		}
		return state, nil

	case "NetworkStateFromText":
		var statusText, wifiText string
		var ipByDevice map[string]string
		if err := args(c, &statusText, &wifiText, &ipByDevice); err != nil {
			return nil, err
		}
		return collect.NetworkStateFromText(statusText, wifiText, ipByDevice), nil

	case "ParseHistoryItems":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.ParseHistoryItems(text), nil

	case "FormatAge":
		var seconds float64
		if err := args(c, &seconds); err != nil {
			return nil, err
		}
		return collect.FormatAge(seconds), nil

	case "NotificationsStateFromParts":
		var items []collect.HistoryItem
		var pausedText string
		var nowUS float64
		var collapsed []string
		var lastSeen int64
		if err := args(c, &items, &pausedText, &nowUS, &collapsed, &lastSeen); err != nil {
			return nil, err
		}
		collapsedSet := map[string]bool{}
		for _, app := range collapsed {
			collapsedSet[app] = true
		}
		return collect.NotificationsStateFromParts(items, pausedText, nowUS, collapsedSet, lastSeen), nil

	case "PathStem":
		var path string
		if err := args(c, &path); err != nil {
			return nil, err
		}
		return collect.PathStem(path), nil

	case "WallpaperItems":
		var files []string
		var current string
		var realBy map[string]string
		if err := args(c, &files, &current, &realBy); err != nil {
			return nil, err
		}
		// realBy stands in for Path.resolve(); the test supplies the mapping so
		// neither side touches the filesystem.
		realPath := func(p string) string {
			if mapped, ok := realBy[p]; ok {
				return mapped
			}
			return p
		}
		thumb := func(p string) string { return "thumb:" + p }
		return collect.WallpaperItems(files, current, realPath, thumb), nil

	case "RowsFromItems":
		var items []collect.WallpaperItem
		var columns int
		if err := args(c, &items, &columns); err != nil {
			return nil, err
		}
		return collect.RowsFromItems(items, columns), nil

	case "WorkspaceStateFromJSON":
		var activeJSON, workspacesJSON, clientsJSON string
		if err := args(c, &activeJSON, &workspacesJSON, &clientsJSON); err != nil {
			return nil, err
		}
		return collect.WorkspaceStateFromJSON(activeJSON, workspacesJSON, clientsJSON), nil

	case "MissingBarMonitors":
		var monitorsJSON, activeWindows string
		if err := args(c, &monitorsJSON, &activeWindows); err != nil {
			return nil, err
		}
		return collect.MissingBarMonitors(monitorsJSON, activeWindows), nil

	case "ParseAwwwQuery":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.ParseAwwwQuery(text), nil

	case "IsInternalMonitor":
		var name string
		if err := args(c, &name); err != nil {
			return nil, err
		}
		return collect.IsInternalMonitor(name), nil

	case "SplitMonitors":
		var monitors []map[string]any
		if err := args(c, &monitors); err != nil {
			return nil, err
		}
		internal, external := collect.SplitMonitors(monitors)
		return []any{internal, external}, nil

	case "VolumeEventIsRelevant":
		var line string
		if err := args(c, &line); err != nil {
			return nil, err
		}
		return collect.VolumeEventIsRelevant(line), nil
	}
	return nil, fmt.Errorf("unknown fn: %s", c.Fn)
}

func args(c call, targets ...any) error {
	if len(c.Args) != len(targets) {
		return fmt.Errorf("%s: want %d args, got %d", c.Fn, len(targets), len(c.Args))
	}
	for i, target := range targets {
		// UseNumber, matching how the daemon will decode: it keeps 5 distinct
		// from 5.0 for str(), and keeps every number a json.Number so the
		// truthiness and equality helpers see one type rather than two.
		decoder := json.NewDecoder(bytes.NewReader(c.Args[i]))
		decoder.UseNumber()
		if err := decoder.Decode(target); err != nil {
			return fmt.Errorf("%s arg %d: %w", c.Fn, i, err)
		}
	}
	return nil
}
