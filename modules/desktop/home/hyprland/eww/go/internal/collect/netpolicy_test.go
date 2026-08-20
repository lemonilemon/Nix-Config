package collect

import (
	"strings"
	"testing"
	"time"
)

// probeWorld installs a fake network for one probe: which addresses accept a
// TCP connection, and which names resolve.
func probeWorld(t *testing.T, dialable map[string]bool, resolves map[string][]string) {
	t.Helper()
	savedDial, savedLookup := DialTCP, LookupHost
	DialTCP = func(address string, _ time.Duration) bool { return dialable[address] }
	LookupHost = func(name string) ([]string, error) {
		if addresses, ok := resolves[name]; ok && len(addresses) > 0 {
			return addresses, nil
		}
		return nil, errValueLike("no such host: " + name)
	}
	t.Cleanup(func() { DialTCP, LookupHost = savedDial, savedLookup })
}

type errValueLike string

func (e errValueLike) Error() string { return string(e) }

// A healthy network: everything reachable, DNS telling the truth.
func healthyWorld(t *testing.T) {
	probeWorld(t,
		map[string]bool{
			netPolicyBaseline: true,
			netPolicyIPv6:     true,
			"140.82.121.4:22": true,
		},
		map[string][]string{
			netPolicySSHHost:  {"140.82.121.4"},
			netPolicyRealHost: {"104.16.133.229"},
		})
}

func TestProbeReportsAHealthyNetwork(t *testing.T) {
	healthyWorld(t)
	record, _ := ProbeNetPolicy()
	if record.SSH != "ok" || record.DNS != "ok" || record.IPv6 != "ok" {
		t.Errorf("healthy network reported %+v", record)
	}
	if record.ProbedAt <= 0 {
		t.Error("record carries no timestamp, so its age could never be shown")
	}
}

// The case the whole feature exists for.
func TestProbeDetectsABlockedSSHPort(t *testing.T) {
	probeWorld(t,
		map[string]bool{netPolicyBaseline: true, "140.82.121.4:22": false},
		map[string][]string{
			netPolicySSHHost: {"140.82.121.4"}, netPolicyRealHost: {"1.2.3.4"},
		})
	if record, _ := ProbeNetPolicy(); record.SSH != "blocked" {
		t.Errorf("SSH reported %q on a network that refuses port 22", record.SSH)
	}
}

// A resolver that answers a name RFC 2606 guarantees cannot exist is rewriting
// answers. This outranks "DNS works", because it does work -- it just lies.
func TestProbeDetectsDNSHijacking(t *testing.T) {
	probeWorld(t,
		map[string]bool{netPolicyBaseline: true, "140.82.121.4:22": true},
		map[string][]string{
			netPolicySSHHost:     {"140.82.121.4"},
			netPolicyRealHost:    {"104.16.133.229"},
			netPolicyInvalidHost: {"192.0.2.1"}, // the portal's own page
		})
	if record, _ := ProbeNetPolicy(); record.DNS != "hijacked" {
		t.Errorf("DNS reported %q where a reserved name resolved", record.DNS)
	}
}

func TestProbeDistinguishesBrokenDNSFromHijackedDNS(t *testing.T) {
	probeWorld(t,
		map[string]bool{netPolicyBaseline: true},
		map[string][]string{}) // nothing resolves at all
	record, _ := ProbeNetPolicy()
	if record.DNS != "broken" {
		t.Errorf("DNS reported %q where nothing resolves", record.DNS)
	}
	// And SSH must not be called blocked when the name could not be looked up:
	// a resolver failure says nothing about port 22.
	if record.SSH != "unknown" {
		t.Errorf("SSH reported %q when its hostname would not resolve", record.SSH)
	}
}

// Without the baseline, a dead link would report every port as blocked and send
// the reader hunting for a firewall that is not there.
func TestProbeBlamesNothingWhenTheLinkIsDead(t *testing.T) {
	probeWorld(t, map[string]bool{}, map[string][]string{})
	record, conclusive := ProbeNetPolicy()
	// And says so, so EnsureNetPolicy declines to cache it. A link mid-DHCP
	// recorded as "SSH blocked" would never be re-probed.
	if conclusive {
		t.Error("a dead link reported a conclusive result, which would be cached forever")
	}
	for name, got := range map[string]string{
		"ssh": record.SSH, "dns": record.DNS, "ipv6": record.IPv6,
	} {
		if got != "unknown" {
			t.Errorf("%s reported %q on a link carrying nothing, want unknown", name, got)
		}
	}
}

func TestProbeReportsIPv4OnlyNetworks(t *testing.T) {
	probeWorld(t,
		map[string]bool{netPolicyBaseline: true, "140.82.121.4:22": true},
		map[string][]string{
			netPolicySSHHost: {"140.82.121.4"}, netPolicyRealHost: {"1.2.3.4"},
		})
	if record, _ := ProbeNetPolicy(); record.IPv6 != "absent" {
		t.Errorf("IPv6 reported %q with no v6 route", record.IPv6)
	}
}

func policyRecord(ssh, dns, ipv6 string) NetPolicyRecord {
	return NetPolicyRecord{Name: "Tenda_5G", SSH: ssh, DNS: dns, IPv6: ipv6, ProbedAt: savedEpoch}
}

// The line names one consequence, and which one is the whole content of the
// ordering. Everything not chosen is still in the tooltip.
func TestCaptionLeadsWithTheWorstFinding(t *testing.T) {
	for _, c := range []struct {
		name        string
		record      NetPolicyRecord
		want, class string
	}{
		{"hijacked lookups outrank a blocked port",
			policyRecord("blocked", "hijacked", "absent"),
			"Domain lookups are being redirected", "alert"},
		{"dead lookups are the loudest thing there is",
			policyRecord("ok", "broken", "ok"),
			"Domain lookups aren't working", "bad"},
		{"a blocked port outranks missing IPv6",
			policyRecord("blocked", "ok", "absent"),
			"SSH and git push are blocked here", "alert"},
		{"a clean network says what was confirmed, quietly",
			policyRecord("ok", "ok", "ok"),
			"SSH and lookups work here", "quiet"},
		{"missing IPv6 is a suffix, never the headline",
			policyRecord("ok", "ok", "absent"),
			"SSH and lookups work here · IPv4 only", "quiet"},
	} {
		got, class := netPolicyCaption(c.record)
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if class != c.class {
			t.Errorf("%s: class %q, want %q", c.name, class, c.class)
		}
	}
}

// An absent IPv6 route is the ordinary case on most networks. Colouring it as a
// fault would cry wolf on nearly everything anyone joins -- which is exactly
// what the chip row this replaced did.
func TestMissingIPv6IsNeverAnAlert(t *testing.T) {
	_, class := netPolicyCaption(policyRecord("ok", "ok", "absent"))
	if class != "quiet" {
		t.Errorf("a v4-only network renders as %q, want quiet", class)
	}
}

// The findings the caption did not choose still have to be reachable.
func TestTooltipKeepsEveryFinding(t *testing.T) {
	pinClock(t, sameDayLaterEpoch)
	detail := netPolicyDetail(policyRecord("blocked", "hijacked", "absent"), 0)
	for _, want := range []string{"blocked", "redirected", "not available", "Checked 14:32"} {
		if !strings.Contains(detail, want) {
			t.Errorf("tooltip %q does not mention %q", detail, want)
		}
	}
}

// Same rule the speed card enforces: an answer belongs to the connection it was
// taken on, and a network with no record of its own shows nothing.
func TestPolicyCardDoesNotBorrowAnotherNetworksAnswers(t *testing.T) {
	pinClock(t, sameDayLaterEpoch)

	known := NetPolicyCardFrom(policyRecord("blocked", "ok", "ok"), true, false, "wifi", 0)
	if known.SSH != "blocked" || known.Status != "ready" {
		t.Fatalf("a record for this network did not render: %+v", known)
	}

	fresh := NetPolicyCardFrom(NetPolicyRecord{}, false, false, "wifi", 0)
	if fresh.SSH != "unknown" || fresh.Status != "none" {
		t.Errorf("another network's answers surfaced: %+v", fresh)
	}
	if !strings.Contains(fresh.Caption, "Not checked") {
		t.Errorf("caption %q does not say this network is unchecked", fresh.Caption)
	}
}

func TestPolicyCardShowsProbeInFlight(t *testing.T) {
	card := NetPolicyCardFrom(policyRecord("ok", "ok", "ok"), true, true, "wifi", 0)
	if card.Status != "probing" {
		t.Errorf("status is %q while a probe is running", card.Status)
	}
	// The stale answers must not sit there looking current while it runs.
	if card.SSH != "unknown" {
		t.Errorf("a running probe kept the previous answer %q", card.SSH)
	}
}

func TestPolicyCardOffline(t *testing.T) {
	card := NetPolicyCardFrom(policyRecord("ok", "ok", "ok"), true, false, "disconnected", 0)
	if card.Status != "offline" {
		t.Errorf("status is %q with no link", card.Status)
	}
}

// A probe is cheap but not free, and policy does not drift. Asking twice for the
// same connection is waste; asking while one is in flight is a race.
func TestEnsureProbesEachConnectionOnce(t *testing.T) {
	withEmptyPolicyCache(t)
	// The machine is still on uuid-a when the probe lands, so the result is
	// filed rather than discarded.
	identityFixture(t, "uuid-a", "wlo1")

	done := make(chan struct{})
	if !EnsureNetPolicy("uuid-a", "Tenda_5G", "full", func() {}, func() { close(done) }) {
		t.Fatal("the first probe did not start")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the probe never finished")
	}

	if EnsureNetPolicy("uuid-a", "Tenda_5G", "full", func() {}, func() {}) {
		t.Error("a second probe ran for a connection already on file")
	}
	if EnsureNetPolicy("", "", "full", func() {}, func() {}) {
		t.Error("a probe ran with no connection identity to file it against")
	}
}

// The probe needs an identity to file its answer against, and CollectNetworkIdentity
// goes through RunText -- so a test driving EnsureNetPolicy must supply it or the
// landing check will correctly reject its own result.
func identityFixture(t *testing.T, uuid, device string) {
	t.Helper()
	restore := InstallFixture(map[string]string{
		"nmcli\x1f-t\x1f-f\x1fUUID,NAME,TYPE,DEVICE\x1fconnection\x1fshow\x1f--active": uuid + ":Tenda_5G:802-11-wireless:" + device + "\n",
		"ip\x1froute":                    "default via 192.168.0.1 dev " + device + " proto dhcp metric 600\n",
		"dial\x1f" + netPolicyBaseline:   "1",
		"dial\x1f140.82.121.4:22":        "1",
		"lookup\x1f" + netPolicySSHHost:  "140.82.121.4",
		"lookup\x1f" + netPolicyRealHost: "104.16.133.229",
	})
	t.Cleanup(restore)
}

func withEmptyPolicyCache(t *testing.T) {
	t.Helper()
	netPolicyLock.Lock()
	savedRecords, savedProbing := netPolicyRecords, netPolicyProbing
	netPolicyRecords, netPolicyProbing = map[string]NetPolicyRecord{}, ""
	netPolicyLock.Unlock()
	t.Cleanup(func() {
		netPolicyLock.Lock()
		netPolicyRecords, netPolicyProbing = savedRecords, savedProbing
		netPolicyLock.Unlock()
	})
}

// The hotel case, and the reason the connectivity gate exists. Behind a captive
// portal DNS really is hijacked and port 22 really is blocked -- but only until
// you sign in, and a record written now would never be revisited.
func TestNoProbeUntilTheLinkActuallyWorks(t *testing.T) {
	withEmptyPolicyCache(t)
	identityFixture(t, "uuid-a", "wlo1")

	for _, connectivity := range []string{"portal", "limited", "none", "unknown"} {
		if EnsureNetPolicy("uuid-a", "Tenda_5G", connectivity, func() {}, func() {}) {
			t.Errorf("probed a network reporting connectivity %q", connectivity)
		}
	}
	if len(NetPolicyRecordsSnapshot()) != 0 {
		t.Error("something was cached before the link was usable")
	}
}

// An answer belongs to the network it was taken on. The probe takes seconds,
// which is long enough to roam onto another profile.
func TestPolicyResultIsDiscardedIfTheNetworkChanged(t *testing.T) {
	withEmptyPolicyCache(t)
	// The fixture reports the machine sitting on uuid-b when the probe lands.
	identityFixture(t, "uuid-b", "wlo1")

	done := make(chan struct{})
	if !EnsureNetPolicy("uuid-a", "Tenda_5G", "full", func() {}, func() { close(done) }) {
		t.Fatal("the probe did not start")
	}
	select {
	case <-done:
		t.Fatal("a result taken while roaming was persisted under the network it started on")
	case <-time.After(2 * time.Second):
	}
	if _, cached := NetPolicyRecordsSnapshot()["uuid-a"]; cached {
		t.Error("a result that landed on another network was filed under uuid-a")
	}
}
