package collect

import (
	"context"
	"net"
	"sync"
	"time"
)

// What the network will and will not LET you do, as opposed to how fast it does it.
//
// Cached per connection like the speed record, and worth far more here: policy is
// a firewall configuration that holds for years where throughput is stale in days.
//
// Two things that belong to this question are deliberately NOT probed, because
// measuring them honestly is not possible from one end:
//
//   - WireGuard and other UDP tunnels. UDP has no handshake, so a dial succeeds
//     locally whether or not anything survives the first hop.
//   - Portal re-authentication intervals. That is an observation over time, and
//     needs the connectivity watcher rather than anything this file can ask.

const (
	// Applied to every check, DNS included -- the resolver seam below carries
	// this as a context deadline, because net.LookupHost has none of its own.
	netPolicyTimeout = 2500 * time.Millisecond

	// The control. An IP literal rather than a hostname on purpose: this is the
	// one check that must not depend on DNS, which is separately under test.
	netPolicyBaseline = "1.1.1.1:443"

	// A real SSH endpoint that is always listening, reached by name: if the name
	// will not resolve the result is "unknown" rather than "blocked", because a
	// DNS failure says nothing about port 22.
	netPolicySSHHost = "github.com"
	netPolicySSHPort = "22"

	// Cloudflare's resolver over IPv6. A v4-only network fails this instantly
	// with "network unreachable" rather than timing out.
	netPolicyIPv6 = "[2606:4700:4700::1111]:443"

	// A name that must never resolve. RFC 2606 reserves .invalid precisely so that
	// it cannot, so an address coming back means something is rewriting answers.
	//
	// Testing a name that SHOULD resolve and comparing addresses would not work:
	// anycast and CDNs legitimately hand different answers to different resolvers.
	netPolicyInvalidHost = "probe-a4f19c73.invalid"

	// And a name that should resolve, to tell "DNS is lying" from "DNS is dead".
	netPolicyRealHost = "cloudflare.com"

	// How many resolved addresses to try before calling a port blocked.
	netPolicySSHAddresses = 3
)

// The two network seams, package-level for the same reason RunText is: a test swaps
// them and InstallFixture drives both at once.
//
// They exist mainly so the suite cannot reach the network -- a probe that dialled
// github.com from `go test` would walk straight through that rail.
var (
	DialTCP = func(address string, timeout time.Duration) bool {
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}

	// Deadline-bearing, unlike net.LookupHost. Without it a resolver that never
	// answers blocks the probe's WaitGroup forever, and since netPolicyProbing is
	// only cleared after that wait, the guard would refuse to probe anything again.
	LookupHost = func(name string) ([]string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), netPolicyTimeout)
		defer cancel()
		return net.DefaultResolver.LookupHost(ctx, name)
	}
)

// NetPolicy is the readout: one line and a tooltip rather than a row of status
// chips.
//
// Caption names a CONSEQUENCE ("SSH and git push are blocked here") rather than a
// mechanism, and Class decides how loudly. Detail carries the three raw findings
// into the tooltip.
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

// ProbeNetPolicy runs every check concurrently. ok=false means the probe was
// INCONCLUSIVE, not that the network is hostile: if a plain HTTPS connection to an
// IP address fails, the link is not carrying traffic at all, and recording that as
// "SSH blocked, DNS broken" would pin a verdict with nothing to ever retry it.
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
		// Every address, not just the first. A v4-only network handed an AAAA record
		// first would otherwise report the port blocked when it is merely the wrong
		// address family, and this machine is v4-only.
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

// NetPolicyCardFrom follows the speed card's rule: a result belongs to the
// connection it was taken on, so a network with no record of its own says so.
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
// Only ever ONE finding, and the ordering is the whole content of this function: a
// hijacked resolver outranks a blocked port, which outranks missing IPv6. IPv6
// never becomes the headline -- on most networks it is absent and nothing cares.
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

// netPolicyDetail is the tooltip: every finding, plus when it was taken. Padded
// into columns because the bar renders in a monospaced font.
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

// EnsureNetPolicy probes this connection if it has never been probed. Automatic,
// unlike the speed test, because it moves a few kilobytes and finishes in under
// three seconds.
//
// publish is called when the probe starts and again when it lands; persist once,
// after the record is in the cache. Two callbacks rather than a return value,
// because this returns as soon as the probe is launched.
//
// connectivity gates the whole thing, and that gate is what keeps the cache honest:
// behind a captive portal DNS really is hijacked and port 22 really is blocked, but
// only until you sign in -- and since the record would then exist, nothing would
// ever re-probe it. Policy does not drift, so a record is never refreshed on a
// timer, which is exactly why it must not be written until it means something.
//
// Returns false when there is nothing to do: no identity, a link not carrying
// traffic yet, a probe already in flight, or an answer already on file.
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

		// The identity is re-read here for the reason RunSpeedtest re-reads it: the
		// probe takes a couple of seconds, long enough to roam onto another profile,
		// and filing these answers under the network we started on would poison it.
		landed := CollectNetworkIdentity()

		netPolicyLock.Lock()
		netPolicyProbing = ""
		if conclusive && landed.UUID == uuid {
			netPolicyRecords[uuid] = record
			netPolicyRecords = PruneNetPolicyRecords(netPolicyRecords, netPolicyRecordLimit)
		}
		netPolicyLock.Unlock()

		publish()
		// Nothing to write when the probe was inconclusive or landed elsewhere; the
		// next tick will try again.
		if conclusive && landed.UUID == uuid {
			persist()
		}
	}()
	return true
}
