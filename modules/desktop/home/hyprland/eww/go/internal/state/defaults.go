package state

import "ewwbar/internal/collect"

// Default is the bar's initial state. Keep it in step with eww.yuck's :initial
// literal; the eww-backend flake check fails the build on drift.
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
		AiUsage: collect.AiUsageDefault(),
		CPU:     "\uF2DB --%",
		Memory: collect.Module{
			Text: "", Tooltip: "", Class: "",
		},
		Temperature: collect.Temperature{Text: "\uF2CB --\u00B0C", Class: ""},
		Network: collect.NetworkStateFull{
			Text: "", Tooltip: "",
			Class: "", WifiEnabled: "false",
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
