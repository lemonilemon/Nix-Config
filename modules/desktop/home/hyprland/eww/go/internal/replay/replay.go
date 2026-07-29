// Package replay answers one recorded call against the ported collectors.
//
// It is the shared body of two things: the golden replay test, and the diffgen
// command, which the eww-backend flake check drives to produce the default
// snapshot. During the port a third caller drove diffgen from Python and diffed
// the two implementations line by line; that gate is gone with the Python.
//
// Outliving the gate is the point. The differential question -- "does Go agree
// with CPython?" -- stopped being answerable the moment the Python was deleted.
// What replaces it is "does Go still answer what it answered when we checked it
// against CPython?", and that needs exactly these inputs and the recorded
// outputs, which is why this dispatch is a package rather than a main.
package replay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"ewwbar/internal/collect"
	"ewwbar/internal/control"
	"ewwbar/internal/pyjson"
	"ewwbar/internal/state"
)

// Call is one recorded invocation.
type Call struct {
	Fn   string            `json:"fn"`
	Args []json.RawMessage `json:"args"`
}

// Answer runs one call against the ported code.
func Answer(c Call) (any, error) {
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
	// in for the subprocess. It is what makes these cases replayable at all: the
	// recorded answer is a function of the fixture, not of the machine, so a
	// collector that shells out to nmcli returns the same thing on a laptop with
	// no network as it did on the machine that recorded it.

	case "CollectActiveWindow", "CollectWorkspace", "CollectMedia",
		"CollectTrayCount", "CollectBluetooth", "CollectVolume",
		"NetworkRadioEnabled", "CollectLinkFallback", "CollectNetworkConnection",
		"CollectNetwork",
		"ClaudeCredentialsPaths", "ClaudeLoadOAuth", "ClaudeQuotaState",
		"OpenusageQuotaStates", "QuotaStates", "QuotaStatesSequence",
		"AiUsageState", "AiUsageStateStaleSequence", "AiUsageStateGoodTwice",
		"RefreshAiUsage",
		"CollectNotifications", "NotifAction", "NotifToggleGroupTwice",
		"ScanWallpaperFiles", "ThumbCachePath", "EnsureThumbnail",
		"CollectWallpaper", "SetWallpaper", "NotifMarkSeenRewind", "ControlHandle",
		"IdleInhibitedState", "LidInhibitedState", "ToggleIdleInhibited",
		"SetIdleInhibited", "SetLidInhibited", "ReadDisplayMode",
		"WriteDisplayMode", "DisplayState", "MonitorState", "SetDisplayMode",
		"AiRefreshCycleProbeTicks":
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

	// -- AI usage: periods and assembly ------------------------------------

	case "AgentDisplayName":
		var key any
		if err := args(c, &key); err != nil {
			return nil, err
		}
		return collect.AgentDisplayName(key), nil

	case "CurrentPeriodKey":
		var kind string
		var rows []map[string]any
		var now float64
		if err := args(c, &kind, &rows, &now); err != nil {
			return nil, err
		}
		return collect.CurrentPeriodKey(kind, rows, now), nil

	case "SyntheticPeriodRow":
		var kind string
		var rows []map[string]any
		var now float64
		if err := args(c, &kind, &rows, &now); err != nil {
			return nil, err
		}
		return collect.SyntheticPeriodRow(kind, rows, now), nil

	case "SelectPeriodRow":
		var rows []any
		var kind string
		var now float64
		if err := args(c, &rows, &kind, &now); err != nil {
			return nil, err
		}
		return collect.SelectPeriodRow(rows, kind, now), nil

	case "PeriodRangeLabel":
		var kind string
		var period any
		var now float64
		if err := args(c, &kind, &period, &now); err != nil {
			return nil, err
		}
		return collect.PeriodRangeLabel(kind, period, now), nil

	case "PeriodAgents":
		var row map[string]any
		if err := args(c, &row); err != nil {
			return nil, err
		}
		return collect.PeriodAgents(row), nil

	case "PeriodState":
		var rows []any
		var kind, label string
		var now float64
		if err := args(c, &rows, &kind, &label, &now); err != nil {
			return nil, err
		}
		return collect.PeriodState(rows, kind, label, now), nil

	case "ApplyQuotas":
		var state collect.AiUsage
		var quotas []collect.Quota
		if err := args(c, &state, &quotas); err != nil {
			return nil, err
		}
		return collect.ApplyQuotas(state, quotas), nil

	case "AiUsageStateFromJSON":
		var reportJSON string
		var now float64
		if err := args(c, &reportJSON, &now); err != nil {
			return nil, err
		}
		return collect.AiUsageStateFromJSON(reportJSON, now), nil

	case "AiUsageDefault":
		return collect.AiUsageDefault(), nil
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
func withFixture(c Call) (any, error) {
	if len(c.Args) < 1 {
		return nil, fmt.Errorf("%s: want a fixture map", c.Fn)
	}
	fixture := map[string]string{}
	if err := json.Unmarshal(c.Args[0], &fixture); err != nil {
		return nil, fmt.Errorf("%s fixture: %w", c.Fn, err)
	}

	restore := collect.InstallFixture(fixture)
	defer restore()

	// Every process-global cache, cleared: a stale entry from a previous case
	// would make this one pass for the wrong reason.
	collect.ResetVolumeSinksCache()
	collect.ResetQuotaCache()
	collect.ResetAiUsageCache()

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

	// -- AI usage: the impure shell ----------------------------------------

	case "ClaudeCredentialsPaths":
		return collect.ClaudeCredentialsPaths(), nil

	case "ClaudeLoadOAuth":
		oauth, ok := collect.ClaudeLoadOAuth()
		if !ok {
			return nil, nil // Python's None
		}
		return oauth, nil

	case "ClaudeQuotaState":
		return collect.ClaudeQuotaState(), nil

	case "OpenusageQuotaStates":
		var now float64
		if len(c.Args) > 1 {
			if err := json.Unmarshal(c.Args[1], &now); err != nil {
				return nil, err
			}
		}
		return collect.OpenusageQuotaStates(now), nil

	case "QuotaStates":
		var refresh bool
		if len(c.Args) > 1 {
			if err := json.Unmarshal(c.Args[1], &refresh); err != nil {
				return nil, err
			}
		}
		return collect.QuotaStates(refresh), nil

	case "QuotaStatesSequence":
		// Two calls, the second with refresh=false, so the cache is observable
		// at all -- a single call can never show whether it was used.
		first := collect.QuotaStates(true)
		second := collect.QuotaStates(false)
		return []any{first, second}, nil

	case "AiUsageState":
		var refreshQuotas bool
		if len(c.Args) > 1 {
			if err := json.Unmarshal(c.Args[1], &refreshQuotas); err != nil {
				return nil, err
			}
		}
		return collect.AiUsageState(refreshQuotas), nil

	case "AiUsageStateStaleSequence":
		// A good report, then a broken one: the second call must serve the
		// remembered state marked stale rather than the placeholder.
		good := collect.AiUsageState(true)
		collect.RunText = func(_ time.Duration, name string, _ ...string) string {
			return "" // ccusage and the probe both go dark
		}
		stale := collect.AiUsageState(true)
		return []any{good, stale}, nil

	case "AiUsageStateGoodTwice":
		// Two GOOD reports back to back. The stale branch's `source ==
		// "missing"` guard is only observable here: with one good call the
		// remembered state is still empty, and with a good-then-broken pair
		// both the guarded and unguarded versions take the stale path.
		first := collect.AiUsageState(true)
		second := collect.AiUsageState(true)
		return []any{first, second}, nil

	case "RefreshAiUsage":
		var current collect.AiUsage
		var refreshQuotas bool
		if err := json.Unmarshal(c.Args[1], &current); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(c.Args[2], &refreshQuotas); err != nil {
			return nil, err
		}
		published := []collect.AiUsage{}
		result := collect.RefreshAiUsage(current, func(value collect.AiUsage) {
			published = append(published, value)
		}, refreshQuotas)
		return map[string]any{"published": published, "result": result}, nil

	// -- display and inhibitors --------------------------------------------
	//
	// These answer with the side-effect journal beside the return value: what
	// set_display_mode DOES is its whole content, and a comparison of the
	// returned Display alone would pass with the body deleted.

	case "IdleInhibitedState":
		return map[string]any{"result": collect.IdleInhibitedState(),
			"journal": collect.FixtureJournal()}, nil

	case "LidInhibitedState":
		return map[string]any{"result": collect.LidInhibitedState(),
			"journal": collect.FixtureJournal()}, nil

	case "ToggleIdleInhibited":
		value, err := collect.ToggleIdleInhibited()
		return map[string]any{"result": okOrNil(value, err), "error": errText(err),
			"journal": collect.FixtureJournal()}, nil

	case "SetIdleInhibited", "SetLidInhibited":
		var enabled bool
		if err := json.Unmarshal(c.Args[1], &enabled); err != nil {
			return nil, err
		}
		setter := collect.SetIdleInhibited
		if c.Fn == "SetLidInhibited" {
			setter = collect.SetLidInhibited
		}
		value, err := setter(enabled)
		return map[string]any{"result": okOrNil(value, err), "error": errText(err),
			"journal": collect.FixtureJournal()}, nil

	case "ReadDisplayMode":
		return collect.ReadDisplayMode(), nil

	case "WriteDisplayMode":
		var mode string
		if err := json.Unmarshal(c.Args[1], &mode); err != nil {
			return nil, err
		}
		collect.WriteDisplayMode(mode)
		return map[string]any{"result": nil, "journal": collect.FixtureJournal()}, nil

	case "DisplayState":
		var status string
		if err := json.Unmarshal(c.Args[1], &status); err != nil {
			return nil, err
		}
		return collect.DisplayState(status), nil

	case "MonitorState":
		monitors := collect.MonitorState()
		if monitors == nil {
			monitors = []map[string]any{}
		}
		return monitors, nil

	case "SetDisplayMode":
		var action string
		if err := json.Unmarshal(c.Args[1], &action); err != nil {
			return nil, err
		}
		display, err := collect.SetDisplayMode(action)
		result := any(display)
		if err != nil {
			result = nil
		}
		return map[string]any{"result": result, "error": errText(err),
			"journal": collect.FixtureJournal()}, nil

	// -- notification actions and the wallpaper picker ---------------------

	case "CollectNotifications":
		collect.ResetNotifyUIState()
		return collect.CollectNotifications(), nil

	case "NotifAction":
		var action, arg string
		if err := json.Unmarshal(c.Args[1], &action); err != nil {
			return nil, err
		}
		if len(c.Args) > 2 {
			if err := json.Unmarshal(c.Args[2], &arg); err != nil {
				return nil, err
			}
		}
		collect.ResetNotifyUIState()
		return notifAction(action, arg), nil

	case "NotifToggleGroupTwice":
		// Collapse state only shows across two calls: the first collapses the
		// group and the second expands it again.
		var app string
		if err := json.Unmarshal(c.Args[1], &app); err != nil {
			return nil, err
		}
		collect.ResetNotifyUIState()
		first := notifAction("toggle-group", app)
		second := notifAction("toggle-group", app)
		return []any{first, second}, nil

	case "NotifMarkSeenRewind":
		// The badge must never move BACKWARDS, and that takes THREE calls to
		// see. Mark against a recent history, mark again against an older one
		// (where an unguarded update would rewind lastSeen), then read the
		// recent history back: only the third call's `new` count differs, and
		// only because the second either kept or lost the high-water mark.
		var histKeys []string
		if err := json.Unmarshal(c.Args[1], &histKeys); err != nil {
			return nil, err
		}
		collect.ResetNotifyUIState()
		inner := collect.RunText
		swap := func(body string) {
			collect.RunText = func(t time.Duration, name string, argv ...string) string {
				if name == "dunstctl" && len(argv) > 0 && argv[0] == "history" {
					return body
				}
				return inner(t, name, argv...)
			}
		}
		// The last entry is a PLAIN READ, not another mark. mark_seen updates
		// the high-water mark before it reads state back, so a third mark
		// would repair the rewind before anyone could observe it.
		results := []any{}
		for i, body := range histKeys {
			swap(body)
			if i == len(histKeys)-1 {
				results = append(results, map[string]any{
					"result": collect.CollectNotifications(), "error": nil})
				continue
			}
			results = append(results, notifAction("mark-seen", ""))
		}
		return results, nil

	case "ControlHandle":
		// The handler answers with its reply JSON, the side-effect journal, and
		// the state it wrote -- all three, because a reply that is right while
		// the state update is wrong leaves the bar showing stale values with no
		// error anywhere.
		var payload map[string]any
		if err := json.Unmarshal(c.Args[1], &payload); err != nil {
			return nil, err
		}
		collect.ResetNotifyUIState()
		store := state.New()
		reply, handleErr := control.Handle(store, payload)
		if handleErr != nil {
			reply = control.ErrorReply(handleErr.Error())
		}
		encoded, err := pyjson.Encode(reply, true)
		if err != nil {
			return nil, err
		}
		snapshot, err := store.Snapshot()
		if err != nil {
			return nil, err
		}
		return map[string]any{"reply": encoded, "snapshot": snapshot,
			"journal": collect.FixtureJournal()}, nil

	case "ScanWallpaperFiles":
		var dir string
		if err := json.Unmarshal(c.Args[1], &dir); err != nil {
			return nil, err
		}
		return collect.ScanWallpaperFiles(dir), nil

	case "ThumbCachePath":
		var path string
		var mtime, size int64
		if err := json.Unmarshal(c.Args[1], &path); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(c.Args[2], &mtime); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(c.Args[3], &size); err != nil {
			return nil, err
		}
		return collect.ThumbCachePath(path, mtime, size), nil

	case "EnsureThumbnail":
		var path string
		if err := json.Unmarshal(c.Args[1], &path); err != nil {
			return nil, err
		}
		return map[string]any{"result": collect.EnsureThumbnail(path),
			"journal": collect.FixtureJournal()}, nil

	case "CollectWallpaper":
		return collect.CollectWallpaper(), nil

	case "SetWallpaper":
		var path string
		if err := json.Unmarshal(c.Args[1], &path); err != nil {
			return nil, err
		}
		value, err := collect.SetWallpaper(path)
		return map[string]any{"result": okOrNil(value, err), "error": errText(err),
			"journal": collect.FixtureJournal()}, nil

	case "AiRefreshCycleProbeTicks":
		// Which ticks actually run the expensive probe. Observable only across
		// a run of ticks, and only because a non-refreshing tick hits the warm
		// cache instead of shelling out.
		var quotaEvery int64
		var ticks int
		if err := json.Unmarshal(c.Args[1], &quotaEvery); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(c.Args[2], &ticks); err != nil {
			return nil, err
		}
		probed := []bool{}
		inner := collect.RunText
		collect.RunText = func(timeout time.Duration, name string, argv ...string) string {
			if name == "openusage-cli" {
				probed[len(probed)-1] = true
			}
			return inner(timeout, name, argv...)
		}
		cycle := collect.AiRefreshCycle(quotaEvery)
		current := collect.AiUsageDefault()
		for i := 0; i < ticks; i++ {
			probed = append(probed, false)
			current = cycle(current, func(value collect.AiUsage) { current = value })
		}
		return probed, nil
	}
	return nil, fmt.Errorf("unknown fixture fn: %s", c.Fn)
}

// notifAction runs one of the six popup actions and reports it the way the
// Python side does, with the side-effect journal beside the result.
func notifAction(action, arg string) map[string]any {
	var value collect.NotificationsState
	var err error
	switch action {
	case "toggle-group":
		value, err = collect.ToggleGroup(arg)
	case "dismiss":
		value, err = collect.DismissNotification(arg)
	case "clear-group":
		value, err = collect.ClearGroup(arg)
	case "clear-all":
		value = collect.ClearAllNotifications()
	case "dnd-toggle":
		value = collect.ToggleDND()
	case "mark-seen":
		value = collect.MarkSeen()
	}
	return map[string]any{"result": okOrNil(value, err), "error": errText(err),
		"journal": collect.FixtureJournal()}
}

// okOrNil drops a Go zero value on the error path. Python raises there and the
// caller records result=None; returning "" would be a harness difference
// reported as a port difference.
func okOrNil(value any, err error) any {
	if err != nil {
		return nil
	}
	return value
}

// errText renders an error the way the Python side reports one: the message
// alone, or nil when there was none.
func errText(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

func args(c Call, targets ...any) error {
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
