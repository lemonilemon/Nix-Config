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
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"ewwbar/internal/collect"
	"ewwbar/internal/pyjson"
	"ewwbar/internal/state"
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

	case "EncodeJSON":
		// The argument is decoded to the same generic shapes the daemon will
		// hold, then re-encoded, so this compares the ENCODER against
		// json.dumps rather than a Go struct against a Python dict.
		if len(c.Args) != 2 {
			return nil, fmt.Errorf("EncodeJSON: want 2 args")
		}
		var ensureASCII bool
		if err := json.Unmarshal(c.Args[1], &ensureASCII); err != nil {
			return nil, err
		}
		// DecodeOrdered, not Unmarshal: object key order is the property being
		// checked, and a map would discard it before the encoder ever saw it.
		value, err := pyjson.DecodeOrdered(c.Args[0])
		if err != nil {
			return nil, err
		}
		encoded, err := pyjson.Encode(value, ensureASCII)
		if err != nil {
			return nil, err
		}
		return encoded, nil

	case "EncodeFloat":
		var f float64
		var ensureASCII bool
		if err := args(c, &f, &ensureASCII); err != nil {
			return nil, err
		}
		return pyjson.Encode(f, ensureASCII)

	case "EncodeNilSlice":
		// There is no way to send a Go nil slice over the wire, so build one
		// here: the point is that it must render as [] and not null.
		var nilSlice []string
		return pyjson.Encode(struct {
			Sinks  []string `json:"sinks"`
			Groups []string `json:"groups"`
		}{Sinks: nilSlice, Groups: []string{}}, false)

	// --- impure collectors, driven from a command -> output fixture ---------
	//
	// The first argument is a map keyed by the argv joined with \x1f, standing
	// in for run_text. The Python side patches collectors.run_text with the
	// same map, so both implementations see identical subprocess output and any
	// difference is theirs, not the machine's.

	case "CollectActiveWindow", "CollectWorkspace", "CollectMedia",
		"CollectTrayCount", "CollectBluetooth", "CollectVolume",
		"NetworkRadioEnabled", "CollectLinkFallback", "CollectNetworkConnection",
		"CollectNetwork":
		return withFixture(c)

	case "CollectVolumeSequence":
		// Two calls under two fixtures, sharing one warm cache. The second
		// fixture has no pactl output at all, so if the cache is working the
		// sinks survive and if it is not they vanish -- which single-call
		// testing cannot distinguish, because the cache starts cold every time.
		var first, second map[string]string
		if err := args(c, &first, &second); err != nil {
			return nil, err
		}
		savedRun := collect.RunText
		defer func() { collect.RunText = savedRun }()
		collect.ResetVolumeSinksCache()

		fixtureRunner := func(fixture map[string]string) func(time.Duration, string, ...string) string {
			return func(_ time.Duration, name string, argv ...string) string {
				return fixture[strings.Join(append([]string{name}, argv...), "\x1f")]
			}
		}
		collect.RunText = fixtureRunner(first)
		a := collect.CollectVolume(true)
		collect.RunText = fixtureRunner(second)
		b := collect.CollectVolume(false)
		collect.ResetVolumeSinksCache()
		return []any{a, b}, nil

	case "DefaultSnapshot":
		// The bytes BarState() emits before a single collector has run. This is
		// what eww.yuck's :initial literal has to match, and what the whole
		// encoder chain has to reproduce.
		return state.New().Snapshot()

	case "CPUStateFromSamples":
		var first, second string
		if err := args(c, &first, &second); err != nil {
			return nil, err
		}
		return collect.CPUStateFromSamples(first, second), nil

	case "TemperatureStateFromReadings":
		var millidegrees []int64
		if err := args(c, &millidegrees); err != nil {
			return nil, err
		}
		return collect.TemperatureStateFromReadings(millidegrees), nil

	case "BatteryStateFromFiles":
		var files map[string]string
		if err := args(c, &files); err != nil {
			return nil, err
		}
		battery, ok := collect.BatteryStateFromFiles(files)
		if !ok {
			return collect.BatteryDefault(), nil
		}
		return battery, nil

	case "VolumeEventIsRelevant":
		var line string
		if err := args(c, &line); err != nil {
			return nil, err
		}
		return collect.VolumeEventIsRelevant(line), nil

	// -- AI usage: value probing, epoch parsing ---------------------------
	//
	// Anything that can produce a non-finite float is answered as an IEEE bit
	// pattern rather than a number. Go's encoding/json refuses Inf and NaN
	// outright while Python's json emits bare Infinity, so a float("1e999")
	// case would fail in the harness rather than in the code under test. Bits
	// also make the comparison exact for -0.0 and for every NaN payload.
	case "PyFloat":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		value, ok := collect.PyFloat(text)
		return []any{ok, floatBits(value)}, nil

	case "NumberValue":
		var data map[string]any
		var keys []string
		if err := args(c, &data, &keys); err != nil {
			return nil, err
		}
		return floatBits(collect.NumberValue(data, keys...)), nil

	case "ListValue":
		var data map[string]any
		var keys []string
		if err := args(c, &data, &keys); err != nil {
			return nil, err
		}
		return collect.ListValue(data, keys...), nil

	case "ParseISOEpoch":
		var value any
		if err := args(c, &value); err != nil {
			return nil, err
		}
		epoch, ok := collect.ParseISOEpoch(value)
		if !ok {
			return nil, nil // Python's None
		}
		return epoch, nil

	case "FormatClockTime":
		var epoch any
		if err := args(c, &epoch); err != nil {
			return nil, err
		}
		return collect.FormatClockTime(epoch), nil

	case "DailyTokenValues":
		var row map[string]any
		if err := args(c, &row); err != nil {
			return nil, err
		}
		return collect.DailyTokenValues(row), nil

	case "AgentsText":
		var agents []string
		if err := args(c, &agents); err != nil {
			return nil, err
		}
		return collect.AgentsText(agents), nil

	case "PeriodAgentKeys":
		var row map[string]any
		if err := args(c, &row); err != nil {
			return nil, err
		}
		return collect.PeriodAgentKeys(row), nil

	case "PyLower":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.PyLower(text), nil

	case "PyTitle":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.PyTitle(text), nil

	// -- AI usage: the provider quota cards --------------------------------

	case "QuotaWindowClass":
		var percent int
		var hasReset bool
		if err := args(c, &percent, &hasReset); err != nil {
			return nil, err
		}
		return collect.QuotaWindowClass(percent, hasReset), nil

	case "QuotaCardClass":
		var quota collect.Quota
		if err := args(c, &quota); err != nil {
			return nil, err
		}
		return collect.QuotaCardClass(quota), nil

	case "QuotaDefault":
		var key, name, status string
		if err := args(c, &key, &name, &status); err != nil {
			return nil, err
		}
		return collect.QuotaDefault(key, name, status), nil

	case "OpenusageWindow":
		var line map[string]any
		var now float64
		if err := args(c, &line, &now); err != nil {
			return nil, err
		}
		return collect.OpenusageWindow(line, now), nil

	case "OpenusageMeta":
		var line map[string]any
		if err := args(c, &line); err != nil {
			return nil, err
		}
		entry, ok := collect.OpenusageMeta(line)
		if !ok {
			return nil, nil // Python's None
		}
		return entry, nil

	case "QuotaFromOpenusage":
		var snapshot any
		var key, name string
		var now float64
		if err := args(c, &snapshot, &key, &name, &now); err != nil {
			return nil, err
		}
		return collect.QuotaFromOpenusage(snapshot, key, name, now), nil

	case "FormatClaudePlan":
		var value any
		if err := args(c, &value); err != nil {
			return nil, err
		}
		return collect.FormatClaudePlan(value), nil

	case "ClaudeWindowState":
		var label string
		var window any
		var now float64
		if err := args(c, &label, &window, &now); err != nil {
			return nil, err
		}
		return collect.ClaudeWindowState(label, window, now), nil

	case "ClaudeQuotaStateFromJSON":
		var usageJSON string
		var plan any
		var now float64
		if err := args(c, &usageJSON, &plan, &now); err != nil {
			return nil, err
		}
		return collect.ClaudeQuotaStateFromJSON(usageJSON, plan, now), nil
	}
	return nil, fmt.Errorf("unknown fn: %s", c.Fn)
}

// floatBits answers a float as its IEEE bit pattern, canonicalising NaN.
//
// Bits are the comparison currency for anything that can go non-finite,
// because encoding/json refuses Inf and NaN where Python's json emits them
// bare. NaN is canonicalised first: CPython's float("nan") carries payload
// 0x7FF8000000000000 and Go's math.NaN() carries 0x7FF8000000000001, and that
// difference is in the runtimes' choice of quiet NaN, not in the port.
func floatBits(value float64) uint64 {
	if math.IsNaN(value) {
		return 0x7FF8000000000000
	}
	return math.Float64bits(value)
}

// withFixture installs a fake RunText/ReadTextFile for the duration of one
// call, then restores them. Not concurrent-safe, and does not need to be:
// diffgen answers one line at a time.
func withFixture(c call) (any, error) {
	if len(c.Args) < 1 {
		return nil, fmt.Errorf("%s: want a fixture map", c.Fn)
	}
	fixture := map[string]string{}
	if err := json.Unmarshal(c.Args[0], &fixture); err != nil {
		return nil, fmt.Errorf("%s fixture: %w", c.Fn, err)
	}

	savedRun, savedRead := collect.RunText, collect.ReadTextFile
	defer func() { collect.RunText, collect.ReadTextFile = savedRun, savedRead }()

	collect.RunText = func(_ time.Duration, name string, argv ...string) string {
		return fixture[strings.Join(append([]string{name}, argv...), "\x1f")]
	}
	collect.ReadTextFile = func(path string) (string, bool) {
		value, ok := fixture["file\x1f"+path]
		return value, ok
	}

	// The sink cache is process-global; a stale entry from a previous case
	// would make this one pass for the wrong reason.
	collect.ResetVolumeSinksCache()

	switch c.Fn {
	case "CollectActiveWindow":
		return collect.CollectActiveWindow(), nil
	case "CollectWorkspace":
		return collect.CollectWorkspace(), nil
	case "CollectMedia":
		return collect.CollectMedia(), nil
	case "CollectTrayCount":
		return collect.CollectTrayCount(), nil
	case "CollectBluetooth":
		return collect.CollectBluetooth(), nil
	case "CollectVolume":
		var refresh bool
		if len(c.Args) > 1 {
			if err := json.Unmarshal(c.Args[1], &refresh); err != nil {
				return nil, err
			}
		}
		return collect.CollectVolume(refresh), nil
	case "NetworkRadioEnabled":
		return collect.NetworkRadioEnabled(), nil
	case "CollectLinkFallback":
		state, ok := collect.CollectLinkFallback()
		if !ok {
			return nil, nil // Python's None
		}
		return state, nil
	case "CollectNetworkConnection":
		return collect.CollectNetworkConnection(), nil
	case "CollectNetwork":
		return collect.CollectNetwork(), nil
	}
	return nil, fmt.Errorf("unknown fixture fn: %s", c.Fn)
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
