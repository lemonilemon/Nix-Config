// Package state holds the bar's live state and the snapshot eww consumes.
//
// Every type here is a struct rather than a map. That is load-bearing: pyjson
// emits struct fields in declaration order but sorts map keys, and EmitLoop
// decides whether to write by comparing encoded bytes, so the encoding has to be
// a function of the state alone.
//
// The particular order below is the order the Python built each dict in. It no
// longer has to match anything -- eww reads the snapshot by key -- so treat it
// as history rather than a constraint. What must not drift is the CONTENT of
// Default(), which the eww-backend flake check pins against eww.yuck's :initial
// literal.
package state

import "ewwbar/internal/collect"

// Bar is the whole state the daemon emits, in emit order.
type Bar struct {
	ActiveWindow   collect.ActiveWindowState  `json:"active_window"`
	WorkspaceState collect.WorkspaceState     `json:"workspace_state"`
	Submap         string                     `json:"submap"`
	Clock          collect.Clock              `json:"clock"`
	Media          collect.MediaState         `json:"media"`
	AiUsage        collect.AiUsage            `json:"ai_usage"`
	CPU            string                     `json:"cpu"`
	Memory         collect.Module             `json:"memory"`
	Temperature    collect.Temperature        `json:"temperature"`
	Network        collect.NetworkStateFull   `json:"network"`
	Volume         collect.VolumeState        `json:"volume"`
	Battery        collect.Battery            `json:"battery"`
	Bluetooth      collect.BluetoothState     `json:"bluetooth"`
	TrayCount      int                        `json:"tray_count"`
	Notifications  collect.NotificationsState `json:"notifications"`
	Wallpaper      collect.Wallpaper          `json:"wallpaper"`
	IdleInhibited  string                     `json:"idle_inhibited"`
	Display        collect.Display            `json:"display"`
}
