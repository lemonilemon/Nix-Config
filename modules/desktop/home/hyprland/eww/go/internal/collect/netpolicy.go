package collect

import (
	"context"
	"net"
	"sync"
	"time"
)

// What the network will and will not LET you do, as opposed to how fast it does
// it.
//
// This is the question the speed card cannot answer and the one that actually
// ruins an afternoon. Nobody's day is destroyed by 40 Mb/s instead of 80; plenty
// of days are destroyed by a hotel network that silently drops port 22.
//
// It caches per connection for the same reason the speed record does, but the
// caching is worth far more here: policy is a firewall configuration that holds
// for years, where throughput is stale in days. One probe stays true for as long
// as you keep visiting the place.
//
// Two things that belong to this question are deliberately NOT probed, because
// measuring them honestly is not possible from one end:
//
//   - WireGuard and other UDP tunnels. UDP has no handshake, so a dial succeeds
//     locally whether or not anything survives the first hop; telling "blocked"
//     from "working" needs a cooperating server to answer, and guessing from a
//     silent socket would produce a confident wrong answer.
//   - Portal re-authentication intervals. That is an observation over time, not
//     a probe, and it needs the connectivity watcher to notice a flip back to
//     "portal" rather than anything this file can ask.

const (
	// Applied to every check, DNS included -- the resolver seam below carries
	// this as a context deadline, because net.LookupHost has none of its own.
	//
	// Not quite the wall-clock cost of the probe: the checks run concurrently,
	// but the SSH one resolves and then dials, and may dial more than one
	// address, so its worst case is a small multiple of this. The common case
	// is well under a second.
	netPolicyTimeout = 2500 * time.Millisecond

	// The control. An IP literal rather than a hostname on purpose: this is the
	// one check that must not depend on DNS, because DNS is separately under
	// test and a resolver failure would otherwise report every port as blocked.
	netPolicyBaseline = "1.1.1.1:443"

	// A real SSH endpoint that is always listening. Reached by name, which is
	// deliberate -- if the name will not resolve, the result is "unknown"
	// rather than "blocked", because a DNS failure says nothing about port 22.
	netPolicySSHHost = "github.com"
	netPolicySSHPort = "22"

	// Cloudflare's resolver over IPv6. A v4-only network fails this instantly
	// with "network unreachable" rather than timing out, so the common negative
	// case costs nothing.
	netPolicyIPv6 = "[2606:4700:4700::1111]:443"

	// A name that must never resolve. RFC 2606 reserves .invalid precisely so
	// that it cannot, so an address coming back means something between here
	// and the root is rewriting answers -- the NXDOMAIN hijacking that captive
	// portals and some ISPs do.
	//
	// Testing a name that SHOULD resolve and comparing addresses would not work:
	// anycast and CDNs legitimately hand different answers to different
	// resolvers, and every one of those would read as a hijack.
	netPolicyInvalidHost = "probe-a4f19c73.invalid"

	// And a name that should resolve, to tell "DNS is lying" from "DNS is dead".
	netPolicyRealHost = "cloudflare.com"

	// How many resolved addresses to try before calling a port blocked.
	netPolicySSHAddresses = 3
)

// The two network seams, package-level for the same reason RunText is: a test
// swaps them and InstallFixture drives both at once. Nothing in production
// reassigns them.
//
// They exist mainly so the suite cannot reach the network. The control tests
// install an empty fixture specifically so a handler under test cannot touch the
// developer's session, and a probe that dialled github.com from `go test` would
// walk straight through that rail.
var (
	DialTCP = func(address string, timeout time.Duration) bool {
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}

	// Deadline-bearing, unlike net.LookupHost, which has none. Without it a
	// resolver that never answers blocks the probe's WaitGroup forever, and
	// because netPolicyProbing is only cleared after that wait, the guard would
	// then refuse to probe any network for the rest of the session.
	LookupHost = func(name string) ([]string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), netPolicyTimeout)
		defer cancel()
		return net.DefaultResolver.LookupHost(ctx, name)
	}
)

// NetPolicy is the readout, which is one line and a tooltip rather than a row
// of status chips.
//
// The chips came first and were wrong. Three coloured dots labelled SSH, DNS and
// IPv6 need a legend the popup has nowhere to put, spend two thirds of their
// pixels announcing that things work, and -- worst -- rendered the ordinary case
// of a v4-only network as a greyed-out dot beside two green ones, which reads as
// a fault. A network with nothing wrong should say so in one quiet line and
// stop talking.
//
// So Caption is the whole design: it names a CONSEQUENCE ("SSH and git push are
// blocked here") rather than a mechanism, and Class decides how loudly. Detail
// carries the three raw findings into the tooltip, where completeness is free
// because nobody reads a tooltip by accident.
type NetPolicy struct {
	Status  string `json:"status"`
	Class   string `json:"class"`
	Caption string `json:"caption"`
	Detail  string `json:"detail"`
	SSH     string `json:"ssh"`
	DNS     string `json:"dns"`
	IPv6    string `json:"ipv6"`
}

// NetPolicyDefault is the card before anything has been probed.
//
// Keep it in step with eww.yuck's net_policy defvar:
//
//	go run ./internal/collect/diffgen <<< '{"fn":"NetPolicyDefault","args":[]}'
func NetPolicyDefault() NetPolicy {
	return NetPolicy{
		Status: "none", Class: "quiet",
		SSH: "unknown", DNS: "unknown", IPv6: "unknown",
		Caption: "Not checked on this network",
		Detail:  "Nothing has been checked on this network yet",
	}
}

// NetPolicyRecord is one network's probe result, as cached and persisted.
type NetPolicyRecord struct {
	Name     string  `json:"name"`
	SSH      string  `json:"ssh"`
	DNS      string  `json:"dns"`
	IPv6     string  `json:"ipv6"`
	ProbedAt float64 `json:"probed_at"`
}

// ProbeNetPolicy runs every check concurrently and reports what it found.
//
// ok=false means the probe was INCONCLUSIVE, not that the network is hostile,
// and the distinction is the difference between a useful cache and a poisoned
// one. If a plain HTTPS connection to an IP address fails, this link is not
// carrying traffic at all -- mid-DHCP, or a moment of outage -- and recording
// that as "SSH blocked, DNS broken" would pin a verdict on a network that was
// merely not ready yet, with nothing to ever retry it.
func ProbeNetPolicy() (NetPolicyRecord, bool) {
	var (
		baseline, ssh, ipv6 bool
		sshResolved         bool
		dnsHijacked         bool
		dnsWorks            bool
		wait                sync.WaitGroup
	)

	wait.Add(5)
	go func() { defer wait.Done(); baseline = DialTCP(netPolicyBaseline, netPolicyTimeout) }()
	go func() { defer wait.Done(); ipv6 = DialTCP(netPolicyIPv6, netPolicyTimeout) }()
	go func() {
		defer wait.Done()
		addresses, err := LookupHost(netPolicySSHHost)
		if err != nil || len(addresses) == 0 {
			return
		}
		sshResolved = true
		// Every address, not just the first. A v4-only network handed an
		// AAAA record first would otherwise report the port blocked when it is
		// merely the wrong address family -- and this machine is v4-only, so
		// that is the common case rather than the exotic one.
		for _, address := range addresses[:min(len(addresses), netPolicySSHAddresses)] {
			if DialTCP(net.JoinHostPort(address, netPolicySSHPort), netPolicyTimeout) {
				ssh = true
				break
			}
		}
	}()
	go func() {
		defer wait.Done()
		if addresses, err := LookupHost(netPolicyInvalidHost); err == nil && len(addresses) > 0 {
			dnsHijacked = true
		}
	}()
	go func() {
		defer wait.Done()
		if addresses, err := LookupHost(netPolicyRealHost); err == nil && len(addresses) > 0 {
			dnsWorks = true
		}
	}()
	wait.Wait()

	record := NetPolicyRecord{ProbedAt: nowOr(0)}
	if !baseline {
		record.SSH, record.DNS, record.IPv6 = "unknown", "unknown", "unknown"
		return record, false
	}

	switch {
	case !sshResolved:
		record.SSH = "unknown"
	case ssh:
		record.SSH = "ok"
	default:
		record.SSH = "blocked"
	}

	switch {
	case dnsHijacked:
		// Checked before "works", because a resolver that answers everything
		// including names that cannot exist is lying, not working.
		record.DNS = "hijacked"
	case dnsWorks:
		record.DNS = "ok"
	default:
		record.DNS = "broken"
	}

	if ipv6 {
		record.IPv6 = "ok"
	} else {
		record.IPv6 = "absent"
	}
	return record, true
}

// NetPolicyCardFrom renders the readout.
//
// Same rule as the speed card, and for the same reason: a result belongs to the
// connection it was taken on, so a network with no record of its own says so
// rather than showing the last network's answers.
func NetPolicyCardFrom(record NetPolicyRecord, haveRecord, probing bool,
	class string, nowEpoch float64) NetPolicy {
	card := NetPolicyDefault()

	if probing {
		card.Status, card.Class = "probing", "live"
		card.Caption = "Checking what this network allows…"
		card.Detail = "Testing outbound SSH, domain lookups and IPv6"
		return card
	}
	if class == "disconnected" {
		card.Status = "offline"
		card.Caption = "No connection"
		card.Detail = "Nothing to check without a link"
		return card
	}
	if !haveRecord {
		return card
	}

	card.Status = "ready"
	card.SSH, card.DNS, card.IPv6 = record.SSH, record.DNS, record.IPv6
	card.Caption, card.Class = netPolicyCaption(record)
	card.Detail = netPolicyDetail(record, nowEpoch)
	return card
}

// netPolicyCaption says the worst thing, in terms of what it stops you doing.
//
// Only ever ONE finding, and the ordering is the whole content of this function.
// A hijacked resolver outranks a blocked port because it breaks things silently
// and in ways that look like somebody else's fault; a blocked port outranks
// missing IPv6 because missing IPv6 is not a fault at all. Anything not chosen
// is still in the tooltip.
//
// IPv6 never becomes the headline. On most networks it is absent and nothing
// cares, so leading with it would cry wolf on the common case -- it earns a
// four-word suffix on the all-clear line and nothing more.
func netPolicyCaption(record NetPolicyRecord) (string, string) {
	switch {
	case record.DNS == "hijacked":
		return "Domain lookups are being redirected", "alert"
	case record.DNS == "broken":
		return "Domain lookups aren't working", "bad"
	case record.SSH == "blocked":
		return "SSH and git push are blocked here", "alert"
	case record.SSH == "unknown":
		return "Domain lookups work; SSH unverified", "quiet"
	}

	// Nothing wrong. Name what was actually confirmed rather than claiming a
	// general all-clear: three things were tested, not the whole network.
	if record.IPv6 == "absent" {
		return "SSH and lookups work here · IPv4 only", "quiet"
	}
	return "SSH and lookups work here", "quiet"
}

// netPolicyDetail is the tooltip: every finding, plus when it was taken.
//
// Padded into columns because the bar renders in a monospaced font, so this
// lines up, and because a tooltip is read deliberately -- the one place where
// showing all three facts costs nothing.
func netPolicyDetail(record NetPolicyRecord, nowEpoch float64) string {
	ssh := map[string]string{
		"ok": "reachable", "blocked": "blocked", "unknown": "not verified",
	}[record.SSH]
	dns := map[string]string{
		"ok": "normal", "hijacked": "redirected", "broken": "not working",
		"unknown": "not verified",
	}[record.DNS]
	ipv6 := map[string]string{
		"ok": "available", "absent": "not available", "unknown": "not verified",
	}[record.IPv6]

	return "SSH (port 22)   " + ssh +
		"\nDomain lookups  " + dns +
		"\nIPv6            " + ipv6 +
		"\nChecked " + strippedAge(record.ProbedAt, nowEpoch)
}

// strippedAge is SpeedtestAge without its leading verb, so the caller can
// supply its own.
func strippedAge(probedEpoch, nowEpoch float64) string {
	const prefix = "Measured "
	age := SpeedtestAge(probedEpoch, nowEpoch)
	if len(age) > len(prefix) && age[:len(prefix)] == prefix {
		return age[len(prefix):]
	}
	return age
}

var (
	netPolicyLock    sync.Mutex
	netPolicyRecords = map[string]NetPolicyRecord{}
	netPolicyProbing string
)

// SeedNetPolicyRecords fills the cache from a restored snapshot, and like
// SeedSpeedtestRecords only fills an empty one.
func SeedNetPolicyRecords(records map[string]NetPolicyRecord) {
	netPolicyLock.Lock()
	defer netPolicyLock.Unlock()
	if len(netPolicyRecords) > 0 {
		return
	}
	for uuid, record := range records {
		netPolicyRecords[uuid] = record
	}
}

// NetPolicyRecordsSnapshot copies the cache out for persisting.
func NetPolicyRecordsSnapshot() map[string]NetPolicyRecord {
	netPolicyLock.Lock()
	defer netPolicyLock.Unlock()
	out := make(map[string]NetPolicyRecord, len(netPolicyRecords))
	for uuid, record := range netPolicyRecords {
		out[uuid] = record
	}
	return out
}

// NetPolicyCardFor assembles the card for one connection out of the cache.
func NetPolicyCardFor(uuid, class string, nowEpoch float64) NetPolicy {
	netPolicyLock.Lock()
	probing := netPolicyProbing != "" && netPolicyProbing == uuid
	var record NetPolicyRecord
	haveRecord := false
	if uuid != "" {
		record, haveRecord = netPolicyRecords[uuid]
	}
	netPolicyLock.Unlock()
	return NetPolicyCardFrom(record, haveRecord, probing, class, nowEpoch)
}

// EnsureNetPolicy probes this connection if it has never been probed.
//
// Automatic, unlike the speed test, and the difference is the cost: this moves a
// few kilobytes and finishes in under three seconds, where a speed test moves 70
// megabytes and saturates the link. Something that cheap should not need asking
// for -- the answer is wanted at exactly the moment you join an unfamiliar
// network, which is the moment you are least likely to think of pressing a
// button.
//
// publish is called when the probe starts and again when it lands; persist is
// called once, after the record is in the cache. Two callbacks rather than a
// return value the caller waits on, because this returns as soon as the probe is
// launched.
//
// connectivity gates the whole thing, and that gate is what keeps the cache
// honest. NetworkManager's own verdict is the only cheap way to know the link is
// actually usable, and probing before it says "full" is how a hotel network gets
// permanently recorded as hostile: behind a captive portal DNS really is
// hijacked and port 22 really is blocked, but only until you sign in -- and
// since the record would then exist, nothing would ever re-probe it.
//
// Returns false when there is nothing to do: no identity, a link that is not
// carrying traffic yet, a probe already in flight, or an answer already on file.
// Policy does not drift, so a record is never refreshed on a timer -- which is
// exactly why it must not be written until it means something.
func EnsureNetPolicy(uuid, name, connectivity string, publish, persist func()) bool {
	if uuid == "" || connectivity != "full" {
		return false
	}

	netPolicyLock.Lock()
	_, known := netPolicyRecords[uuid]
	if known || netPolicyProbing != "" {
		netPolicyLock.Unlock()
		return false
	}
	netPolicyProbing = uuid
	netPolicyLock.Unlock()
	publish()

	go func() {
		record, conclusive := ProbeNetPolicy()
		record.Name = name

		// The identity is re-read here for the reason RunSpeedtest re-reads it:
		// the probe takes a couple of seconds, which is long enough to walk out
		// of range or roam onto another profile, and filing these answers under
		// the network we started on would poison that record permanently.
		landed := CollectNetworkIdentity()

		netPolicyLock.Lock()
		netPolicyProbing = ""
		if conclusive && landed.UUID == uuid {
			netPolicyRecords[uuid] = record
			netPolicyRecords = PruneNetPolicyRecords(netPolicyRecords, netPolicyRecordLimit)
		}
		netPolicyLock.Unlock()

		publish()
		// Nothing to write when the probe was inconclusive or landed elsewhere;
		// the next tick will try again, which is the whole point of not caching
		// it.
		if conclusive && landed.UUID == uuid {
			persist()
		}
	}()
	return true
}
