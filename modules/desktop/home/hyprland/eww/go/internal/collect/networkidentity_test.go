package collect

import "testing"

// An SSID is free text and a connection name is the SSID. Splitting on a bare
// colon would read the UUID out of the wrong column for any network whose name
// contains one, and file that network's speed record under a fragment of its
// own name.
func TestTerseSplitHonoursEscapedColons(t *testing.T) {
	fields := splitNmcliTerse(`uuid-a:Guest\: Lobby:802-11-wireless:wlo1`)
	want := []string{"uuid-a", "Guest: Lobby", "802-11-wireless", "wlo1"}

	if len(fields) != len(want) {
		t.Fatalf("split into %d fields %q, want %d", len(fields), fields, len(want))
	}
	for i := range want {
		if fields[i] != want[i] {
			t.Errorf("field %d is %q, want %q", i, fields[i], want[i])
		}
	}
}

func TestTerseSplitUnescapesBackslashes(t *testing.T) {
	fields := splitNmcliTerse(`uuid-a:Home\\Net:802-11-wireless:wlo1`)
	if fields[1] != `Home\Net` {
		t.Errorf("name is %q, want %q", fields[1], `Home\Net`)
	}
}

const activeConnections = "uuid-eth:Wired connection 1:802-3-ethernet:eno1\n" +
	"uuid-wifi:Tenda_5G:802-11-wireless:wlo1\n" +
	"uuid-lo:lo:loopback:lo\n"

// A docked laptop has several connections up at once. The record has to be
// filed against the one actually carrying traffic, or a wired measurement gets
// remembered as the Wi-Fi network's.
func TestActiveConnectionFollowsTheDefaultRoute(t *testing.T) {
	wifiRoute := "default via 192.168.0.1 dev wlo1 proto dhcp metric 600\n"
	if got := ActiveConnectionFromText(activeConnections, wifiRoute); got.UUID != "uuid-wifi" {
		t.Errorf("picked %+v, want the wlo1 connection", got)
	}

	ethernetRoute := "default via 10.0.0.1 dev eno1 proto dhcp metric 100\n"
	got := ActiveConnectionFromText(activeConnections, ethernetRoute)
	if got.UUID != "uuid-eth" {
		t.Errorf("picked %+v, want the eno1 connection", got)
	}
	if got.Name != "Wired connection 1" {
		t.Errorf("name is %q, want the profile name", got.Name)
	}
}

// Without route text there is still a useful answer, matching the trade
// DefaultRouteDevice's other callers make.
func TestActiveConnectionFallsBackWithoutARoute(t *testing.T) {
	if got := ActiveConnectionFromText(activeConnections, ""); got.UUID != "uuid-eth" {
		t.Errorf("picked %+v, want the first non-loopback connection", got)
	}
}

// Loopback is up on every machine including a disconnected one, where it would
// otherwise win the fallback and become the identity every record is filed
// under -- one bucket collecting every network the laptop ever visits.
func TestActiveConnectionIgnoresLoopback(t *testing.T) {
	if got := ActiveConnectionFromText("uuid-lo:lo:loopback:lo\n", ""); got.UUID != "" {
		t.Errorf("picked %+v, want no identity at all", got)
	}
}

func TestActiveConnectionSurvivesRubbish(t *testing.T) {
	for _, text := range []string{"", "\n\n", "garbage", "a:b"} {
		if got := ActiveConnectionFromText(text, ""); got.UUID != "" {
			t.Errorf("%q produced identity %+v", text, got)
		}
	}
}

// "none" is a claim that there is no internet; "unknown" is an admission that
// nothing was asked. run_text() returns "" for a missing nmcli, a timeout and a
// non-zero exit alike, and the popup must not render any of those as a verdict.
func TestConnectivityDistinguishesUnknownFromNone(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"full\n", "full"},
		{"portal\n", "portal"},
		{"limited\n", "limited"},
		{"none\n", "none"},
		{"", "unknown"},
		{"something else", "unknown"},
	} {
		if got := ConnectivityFromText(c.in); got != c.want {
			t.Errorf("ConnectivityFromText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
