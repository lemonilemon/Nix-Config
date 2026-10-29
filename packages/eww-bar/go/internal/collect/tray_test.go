package collect

import (
	"reflect"
	"strings"
	"testing"
)

func trayStatusKey(name, path string) string {
	return strings.Join([]string{"busctl", "--user", "get-property", name, path,
		"org.kde.StatusNotifierItem", "Status"}, "\x1f")
}

func TestTrayItemsFromTextReadsEveryAddress(t *testing.T) {
	text := `as 3 ":1.37/StatusNotifierItem" ":1.7/org/ayatana/NotificationItem/nm_applet" ":1.9"`
	want := []string{":1.37/StatusNotifierItem", ":1.7/org/ayatana/NotificationItem/nm_applet", ":1.9"}
	if got := TrayItemsFromText(text); !reflect.DeepEqual(got, want) {
		t.Errorf("TrayItemsFromText = %q, want %q", got, want)
	}
	for _, text := range []string{"", "as 0", "ay 2 \"a\""} {
		if got := TrayItemsFromText(text); len(got) != 0 {
			t.Errorf("TrayItemsFromText(%q) = %q, want none", text, got)
		}
	}
}

func TestTrayItemAddressDefaultsAPathlessItem(t *testing.T) {
	if name, path := TrayItemAddress(":1.7/org/ayatana/NotificationItem/nm_applet"); name != ":1.7" || path != "/org/ayatana/NotificationItem/nm_applet" {
		t.Errorf("got %s %s", name, path)
	}
	if name, path := TrayItemAddress(":1.9"); name != ":1.9" || path != "/StatusNotifierItem" {
		t.Errorf("got %s %s", name, path)
	}
}

func TestTrayItemShownHidesOnlyPassive(t *testing.T) {
	cases := map[string]bool{
		`s "Passive"`:        false,
		`s "Active"`:         true,
		`s "NeedsAttention"`: true,
		"":                   true,
	}
	for text, want := range cases {
		if got := TrayItemShownFromText(text); got != want {
			t.Errorf("TrayItemShownFromText(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestCollectTrayCountLeavesOutPassiveItems(t *testing.T) {
	watcher := strings.Join([]string{"busctl", "--user", "get-property",
		"org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher",
		"org.kde.StatusNotifierWatcher", "RegisteredStatusNotifierItems"}, "\x1f")
	restore := InstallFixture(map[string]string{
		watcher: `as 3 ":1.31/StatusNotifierItem" ":1.2530/org/ayatana/NotificationItem/spotify_client" ":1.7/org/ayatana/NotificationItem/nm_applet"`,
		trayStatusKey(":1.31", "/StatusNotifierItem"):                            `s "Active"`,
		trayStatusKey(":1.2530", "/org/ayatana/NotificationItem/spotify_client"): `s "Passive"`,
	})
	defer restore()

	if got := CollectTrayCount(); got != 2 {
		t.Errorf("CollectTrayCount = %d, want 2: the Passive item hidden, the unreadable one shown", got)
	}
}
