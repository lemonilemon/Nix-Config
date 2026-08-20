package state

import "ewwbar/internal/collect"

// Default is the bar's initial state, the value every widget renders before its
// first real reading arrives.
//
// GENERATED ONCE from the Python BarState().snapshot() and committed, not
// generated at build time -- a build-time generator would mean
// import-from-derivation, and this changes about twice a year. Every string is
// written as an escape rather than a literal, because twelve of these fields
// hold Nerd Font glyphs that look exactly like "" in an editor and would not
// survive a copy-paste.
//
// Keep it in step with eww.yuck's :initial literal: the eww-backend flake check
// compares the two and fails the build on drift.
//
// A function, not a var: a shared Go value that one caller mutated would
// corrupt every later reader.
func Default() Bar {
	return Bar{
		ActiveWindow: collect.ActiveWindowState{
			Text: "", Tooltip: "", Class: "",
		},
		WorkspaceState: collect.WorkspaceState{
			Ws1Class: "empty", Ws1Text: "\uF10C",
			Ws2Class: "empty", Ws2Text: "\uF10C",
			Ws3Class: "empty", Ws3Text: "\uF10C",
			Ws4Class: "empty", Ws4Text: "\uF10C",
			Ws5Class: "empty", Ws5Text: "\uF10C",
		},
		Submap: "",
		Clock: collect.Clock{
			Time: "\uF017 --:--", Date: "\uF073 ----", Tooltip: "",
		},
		Media: collect.MediaState{
			Text: "", Status: "", Icon: "\uF001",
			Class: "", Tooltip: "",
		},
		// The whole ai_usage subtree is collectors.AI_USAGE_DEFAULT, which the
		// AI collectors deep-copy on every refresh. One definition, so the two
		// cannot drift -- an earlier attempt at this move silently did not
		// apply, and package state kept a parallel copy that nothing crossed
		// until the control handler did.
		AiUsage: collect.AiUsageDefault(),
		CPU:     "\uF2DB --%",
		Memory: collect.Module{
			Text: "", Tooltip: "", Class: "",
		},
		Temperature: collect.Temperature{Text: "\uF2CB --\u00B0C", Class: ""},
		Network: collect.NetworkStateFull{
			Text: "", Tooltip: "",
			Class: "", WifiEnabled: "false",
			// "unknown" rather than "none": nothing has been asked yet, and
			// the popup renders the two differently on purpose.
			Connectivity: "unknown", ConnUUID: "", ConnName: "",
			Speed:  collect.SpeedtestDefault(),
			Policy: collect.NetPolicyDefault(),
		},
		Volume: collect.VolumeState{
			Text: "", Percent: 0, Muted: "false",
			Class: "", Sinks: []collect.Sink{},
		},
		Battery: collect.Battery{
			Text: "\U000F0084", Alt: "\U000F0084", Capacity: 100,
			Class: "charging", Status: "Unknown", Time: "N/A",
			Health: "--", Power: "\u2014",
		},
		Bluetooth: collect.BluetoothState{
			Text: "", Tooltip: "", Class: "",
			Powered: "false", Devices: []collect.BluetoothDevice{},
		},
		TrayCount: 0,
		Notifications: collect.NotificationsState{
			Paused: "false", New: 0, Count: 0,
			Groups: []collect.NotificationGroup{},
		},
		Wallpaper: collect.Wallpaper{
			Current: "", Count: 0,
			Rows: [][]collect.WallpaperItem{},
		},
		IdleInhibited: "false",
		Display: collect.Display{
			Mode: "normal", LidInhibited: "false",
			Status: "Normal desktop mode", Class: "normal",
		},
	}
}
