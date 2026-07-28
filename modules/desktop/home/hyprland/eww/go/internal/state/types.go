// Package state holds the bar's live state and the snapshot eww consumes.
//
// Every type here is a struct rather than a map, and the field order is the
// order the Python builds each dict in. That is not cosmetic: pyjson emits
// struct fields in declaration order and sorts map keys, and the snapshot has
// to stay byte-identical to what json.dumps produced. state_test.go and
// test_go_equivalence.py both check it against the real thing.
package state

import "ewwbar/internal/collect"

// Clock is collectors.clock_state's shape.
type Clock struct {
	Time    string `json:"time"`
	Date    string `json:"date"`
	Tooltip string `json:"tooltip"`
}

// Display is display.display_state's shape.
type Display struct {
	Mode         string `json:"mode"`
	LidInhibited string `json:"lid_inhibited"`
	Status       string `json:"status"`
	Class        string `json:"class"`
}

// Wallpaper is wallpaper.wallpaper_state's shape.
type Wallpaper struct {
	Current string                    `json:"current"`
	Count   int                       `json:"count"`
	Rows    [][]collect.WallpaperItem `json:"rows"`
}

// AiPeriod is one of ai_usage.periods' three entries.
type AiPeriod struct {
	Label  string   `json:"label"`
	Range  string   `json:"range"`
	Tokens string   `json:"tokens"`
	Cost   string   `json:"cost"`
	Agents []string `json:"agents"`
}

// AiPeriods is ai_usage.periods.
type AiPeriods struct {
	Today AiPeriod `json:"today"`
	Week  AiPeriod `json:"week"`
	Month AiPeriod `json:"month"`
}

// AiMeta is ai_usage.meta.
type AiMeta struct {
	Pricing    string `json:"pricing"`
	Refresh    string `json:"refresh"`
	Status     string `json:"status"`
	Stale      string `json:"stale"`
	Refreshing string `json:"refreshing"`
}

// AiUsage is collectors.ai_usage_state's shape.
type AiUsage struct {
	Text    string          `json:"text"`
	Tooltip string          `json:"tooltip"`
	Class   string          `json:"class"`
	Source  string          `json:"source"`
	Updated string          `json:"updated"`
	Periods AiPeriods       `json:"periods"`
	Agents  string          `json:"agents"`
	Quotas  []collect.Quota `json:"quotas"`
	Meta    AiMeta          `json:"meta"`
}

// Bar is the whole state the daemon emits, in emit order.
type Bar struct {
	ActiveWindow   collect.ActiveWindowState  `json:"active_window"`
	WorkspaceState collect.WorkspaceState     `json:"workspace_state"`
	Submap         string                     `json:"submap"`
	Clock          Clock                      `json:"clock"`
	Media          collect.MediaState         `json:"media"`
	AiUsage        AiUsage                    `json:"ai_usage"`
	CPU            string                     `json:"cpu"`
	Memory         collect.Module             `json:"memory"`
	Temperature    collect.Temperature        `json:"temperature"`
	Network        collect.NetworkStateFull   `json:"network"`
	Volume         collect.VolumeState        `json:"volume"`
	Battery        collect.Battery            `json:"battery"`
	Bluetooth      collect.BluetoothState     `json:"bluetooth"`
	TrayCount      int                        `json:"tray_count"`
	Notifications  collect.NotificationsState `json:"notifications"`
	Wallpaper      Wallpaper                  `json:"wallpaper"`
	IdleInhibited  string                     `json:"idle_inhibited"`
	Display        Display                    `json:"display"`
}
