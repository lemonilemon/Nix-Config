package collect

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

// The speed test behind the network popup's card.
//
// cfspeedtest rather than either speedtest.net client, and the reason is not
// preference. Both speedtest-cli and speedtest-go fail before transferring a
// byte: they fetch https://www.speedtest.net/speedtest-config.php, which now
// answers an unauthenticated GET with a Cloudflare bot challenge -- HTTP 403
// and a "Just a moment..." page, which surfaces as `HTTP Error 403: Forbidden`
// and `XML syntax error on line 1` respectively. Ookla's own client works and
// costs an unfree licence, a GDPR acknowledgement on every run, and a local
// build with no binary cache. speed.cloudflare.com is a public endpoint with no
// such fence, so it is the one that will still work in a year.
//
// The caveat that the number on screen cannot carry: this measures the path to
// the nearest Cloudflare edge, which is peered far better than an average
// route. Optimistic for "is my ISP throttling me", right for "is this hotel
// Wi-Fi usable" -- which is the question the popup exists to answer.
//
// Two invocations rather than one, and the split is what makes the card useful
// while it runs. The probe pass costs a measured 0.7 s and 0.1 MiB and yields
// the round-trip figure; the throughput pass costs 8.7 s and 67.8 MiB. A single
// full run would leave all three fields on placeholders for the whole wait.

const (
	// The light profile. cfspeedtest's own defaults are -n 10 -m 25m, which
	// comes to roughly 361 MB per direction by payload arithmetic; these
	// settle at a measured 33.8 MiB down and 34.0 MiB up in 8.7 s. On a capped
	// hotel line or a phone hotspot that difference is the whole decision, and
	// the extra precision buys nothing for the question being asked.
	speedtestRuns        = "3"
	speedtestPayloadCap  = "10m"
	speedtestLatencyRuns = "10"

	// The probe pass: latency plus the smallest payload cfspeedtest offers.
	// -n 1 and 100k because the download here is only there to keep the
	// invocation well-formed -- the latency block is what is wanted.
	speedtestProbeRuns    = "1"
	speedtestProbePayload = "100k"

	// Timeouts are generous because the payloads are fixed and the link is
	// not: the same 33 MB that moves in 4 s here takes minutes on a bad
	// connection. cfspeedtest skips the larger payload sizes once one exceeds
	// 5 s, which bounds the real worst case well inside this.
	speedtestProbeTimeout = 20 * time.Second
	speedtestTimeout      = 90 * time.Second

	// Rounded up from the measured 67.8 MiB, and stated on the card BEFORE the
	// run rather than after it.
	speedtestDataNote = "about 70 MB"

	// U+2014 em dash. Written as an escape for the reason state.Default() gives:
	// a literal here does not survive review or a copy-paste.
	speedtestDash = "\u2014"

	speedtestNoRecord = "Not measured on this network"
)

// Speedtest is the popup card's readout for whatever network is attached now.
//
// Every field is a rendered string rather than a number, matching the rest of
// the bar's state: eww.yuck does presentation with ternaries and has no
// formatting of its own, so a float reaching it would render as Go's default
// and could not be made to say "86.0".
type Speedtest struct {
	Status  string `json:"status"`
	Down    string `json:"down"`
	Up      string `json:"up"`
	Latency string `json:"latency"`
	Caption string `json:"caption"`
}

// SpeedtestDefault is the card before anything has been measured.
func SpeedtestDefault() Speedtest {
	return Speedtest{
		Status:  "none",
		Down:    speedtestDash,
		Up:      speedtestDash,
		Latency: speedtestDash,
		Caption: speedtestNoRecord,
	}
}

// SpeedtestResult is one measurement in the units the card shows: megabits per
// second, and milliseconds.
type SpeedtestResult struct {
	Down    float64
	Up      float64
	Latency float64
}

// cfSpeedtestReport is the subset of cfspeedtest's `-o json` that is read.
//
// Deliberately not the whole document. The report also carries a metadata block
// holding the machine's public IP address, which has no business in a file this
// writes to disk or in a state blob it prints to stdout.
type cfSpeedtestReport struct {
	Latency struct {
		Avg float64 `json:"avg_latency_ms"`
	} `json:"latency_measurement"`
	Speeds []struct {
		Type    string  `json:"test_type"`
		Payload float64 `json:"payload_size"`
		Avg     float64 `json:"avg"`
	} `json:"speed_measurements"`
}

// SpeedtestFromJSON reads a cfspeedtest report.
//
// The LARGEST payload wins for each direction rather than the mean across all
// of them, and the difference is not marginal: on one link cfspeedtest measured
// 14.9 Mb/s for the 100 kB pass against 85.8 Mb/s for the 10 MB pass, because
// the small transfers are dominated by connection setup. Averaging those
// together would report about a third of the real throughput.
//
// ok=false means the report held nothing worth showing. A report with latency
// and no speed measurements is the probe pass, and is ok.
func SpeedtestFromJSON(text string) (SpeedtestResult, bool) {
	var report cfSpeedtestReport
	if !ParseJSON(text, &report) {
		return SpeedtestResult{}, false
	}

	result := SpeedtestResult{Latency: report.Latency.Avg}
	downPayload, upPayload := -1.0, -1.0
	for _, entry := range report.Speeds {
		switch entry.Type {
		case "Download":
			if entry.Payload > downPayload {
				downPayload, result.Down = entry.Payload, entry.Avg
			}
		case "Upload":
			if entry.Payload > upPayload {
				upPayload, result.Up = entry.Payload, entry.Avg
			}
		}
	}

	if result.Latency <= 0 && downPayload < 0 && upPayload < 0 {
		return SpeedtestResult{}, false
	}
	return result, true
}

// FormatMbps renders a throughput figure for the card's 15px slot.
//
// One decimal below 100 and none above it. The card gives each of its three
// metrics about 60px at that size, which fits "86.0" and not "1024.3" -- and
// above 100 Mb/s the tenths are noise anyway, since two runs on the same link
// disagree by more than that.
func FormatMbps(value float64) string {
	if value <= 0 {
		return speedtestDash
	}
	if value >= 100 {
		return strconv.Itoa(RoundHalfEven(value))
	}
	return strconv.FormatFloat(value, 'f', 1, 64)
}

// FormatLatency renders the round-trip figure, always whole milliseconds.
func FormatLatency(value float64) string {
	if value <= 0 {
		return speedtestDash
	}
	return strconv.Itoa(RoundHalfEven(value))
}

// SpeedtestAge is the card's "when" line.
//
// Follows restoredClock's rule and extends it. That function had two forms
// because a quota card's status line is a memory that must not read as a
// reading; this one has the same job and more room, so it can say "yesterday"
// and "3 days ago" where the other could only say a date.
//
// The boundaries are calendar days, not elapsed hours: a measurement taken at
// 23:50 is "yesterday" at 00:10, because that is what a person means by it.
//
// A record dated in the future -- clock skew, or a restore from a machine with
// a different idea of the time -- falls through to the bare date rather than
// rendering "Measured -2 days ago".
func SpeedtestAge(measuredEpoch, nowEpoch float64) string {
	measured := localTime(measuredEpoch)
	now := localTime(nowOr(nowEpoch))

	measuredDay := time.Date(measured.Year(), measured.Month(), measured.Day(),
		0, 0, 0, 0, measured.Location())
	nowDay := time.Date(now.Year(), now.Month(), now.Day(),
		0, 0, 0, 0, now.Location())
	days := int(nowDay.Sub(measuredDay).Hours() / 24)

	switch {
	case days == 0:
		return "Measured " + measured.Format("15:04")
	case days == 1:
		return "Measured yesterday"
	case days > 1 && days < 7:
		return fmt.Sprintf("Measured %d days ago", days)
	}
	return "Measured " + measured.Format("Jan 2")
}

// SpeedtestView is everything the card builder needs about one moment: the
// stored record for the attached network if there is one, whatever a run in
// flight has produced so far, and the two state strings that can override both.
type SpeedtestView struct {
	Record       SpeedtestRecord
	HaveRecord   bool
	Running      bool
	Partial      SpeedtestResult
	Class        string
	Connectivity string
}

// SpeedtestCardFrom renders the card.
//
// The rule this function exists to enforce: a figure appears only when it was
// measured on the connection attached right now. A record filed under another
// UUID renders as placeholders, NOT as that network's numbers with a caveat
// beside them -- a caveat is the kind of thing a glance misses, and a glance is
// all this card gets. There is deliberately no state for "here is another
// network's speed".
//
// Precedence, in order: a run in flight beats everything, because it is the one
// state the reader caused; no link beats a stored record, because a number
// beside "Disconnected" is a contradiction; a stored record beats the portal
// notice, because having a record proves the portal was signed into before and
// the header and footer already announce the portal loudly.
func SpeedtestCardFrom(view SpeedtestView, nowEpoch float64) Speedtest {
	card := SpeedtestDefault()

	if view.Running {
		card.Status = "running"
		card.Caption = "Testing\u2026 uses " + speedtestDataNote
		// The probe pass lands about eight seconds before the throughput pass,
		// so this field is populated for most of the wait. That is the whole
		// reason the run is split in two.
		if view.Partial.Latency > 0 {
			card.Latency = FormatLatency(view.Partial.Latency)
		}
		return card
	}

	if view.Class == "disconnected" {
		card.Status = "offline"
		card.Caption = "No connection"
		return card
	}

	if view.HaveRecord {
		card.Status = "ready"
		card.Down = FormatMbps(view.Record.Down)
		card.Up = FormatMbps(view.Record.Up)
		card.Latency = FormatLatency(view.Record.Latency)
		card.Caption = SpeedtestAge(view.Record.MeasuredAt, nowEpoch)
		return card
	}

	if view.Connectivity == "portal" {
		card.Status = "blocked"
		card.Caption = "Sign in before testing"
		return card
	}

	return card
}

// The in-memory half of the record store.
//
// Records mirrors what is on disk; running and partial exist only for the
// duration of one run and are never persisted, because "a test was in flight
// when the daemon died" is not a fact worth restoring.
var (
	speedtestLock    sync.Mutex
	speedtestRecords = map[string]SpeedtestRecord{}
	speedtestRunning string
	speedtestPartial SpeedtestResult
)

// SeedSpeedtestRecords fills the cache from a restored snapshot.
//
// Only fills an empty cache, matching SeedQuotaCache: this runs at startup
// beside collectors that may already have written something better, and a
// restore must never overwrite a live reading that won the race.
func SeedSpeedtestRecords(records map[string]SpeedtestRecord) {
	speedtestLock.Lock()
	defer speedtestLock.Unlock()
	if len(speedtestRecords) > 0 {
		return
	}
	for uuid, record := range records {
		speedtestRecords[uuid] = record
	}
}

// SpeedtestRecordsSnapshot copies the cache out for persisting.
func SpeedtestRecordsSnapshot() map[string]SpeedtestRecord {
	speedtestLock.Lock()
	defer speedtestLock.Unlock()
	out := make(map[string]SpeedtestRecord, len(speedtestRecords))
	for uuid, record := range speedtestRecords {
		out[uuid] = record
	}
	return out
}

// SpeedtestCardFor assembles the card for one connection out of the caches.
//
// uuid is the identity of the connection carrying traffic right now. An empty
// uuid can never match a record, which is exactly right: with no identity there
// is no way to know whose numbers a stored record holds.
func SpeedtestCardFor(uuid, class, connectivity string, nowEpoch float64) Speedtest {
	speedtestLock.Lock()
	view := SpeedtestView{
		// Compared against this uuid, not merely non-empty. Otherwise a run
		// started on one network reports "testing" on whatever you switch to,
		// and shows that run's latency beside the new network's name.
		Running:      speedtestRunning != "" && speedtestRunning == uuid,
		Partial:      speedtestPartial,
		Class:        class,
		Connectivity: connectivity,
	}
	if uuid != "" {
		view.Record, view.HaveRecord = speedtestRecords[uuid]
	}
	speedtestLock.Unlock()
	return SpeedtestCardFrom(view, nowEpoch)
}

// RunSpeedtest performs one measurement and files it against the network that
// is attached when it FINISHES.
//
// Re-reading the identity at the end rather than trusting the one captured at
// the start is the whole reason this function is shaped like this. The run
// takes about nine seconds, which is long enough to walk out of the cafe, drop
// onto a phone hotspot, or have NetworkManager roam onto a different profile --
// and filing a hotspot's numbers under the cafe's UUID would poison that record
// permanently, with nothing on screen to suggest it had happened.
//
// publish is called on every state change the card can show: when the run
// starts, when the probe pass lands its latency figure, and when the run ends
// either with a record or without one.
func RunSpeedtest(publish func()) {
	start := CollectNetworkIdentity()
	// Refused rather than attempted. Without an identity the result has nowhere
	// to be filed, so the run would move about 70 MB, show no progress -- the
	// card keys "running" off this uuid -- and discard the answer at the end.
	if start.UUID == "" {
		return
	}

	speedtestLock.Lock()
	speedtestRunning = start.UUID
	speedtestPartial = SpeedtestResult{}
	speedtestLock.Unlock()
	publish()

	defer func() {
		speedtestLock.Lock()
		speedtestRunning = ""
		speedtestPartial = SpeedtestResult{}
		speedtestLock.Unlock()
		publish()
	}()

	if probe, ok := SpeedtestFromJSON(RunText(speedtestProbeTimeout, "cfspeedtest",
		"--download-only",
		"-n", speedtestProbeRuns,
		"-m", speedtestProbePayload,
		"--nr-latency-tests", speedtestLatencyRuns,
		"-o", "json",
	)); ok {
		speedtestLock.Lock()
		speedtestPartial = probe
		speedtestLock.Unlock()
		publish()
	}

	result, ok := SpeedtestFromJSON(RunText(speedtestTimeout, "cfspeedtest",
		"-n", speedtestRuns,
		"-m", speedtestPayloadCap,
		"--nr-latency-tests", speedtestLatencyRuns,
		"-o", "json",
	))
	// Both directions required, where SpeedtestFromJSON is satisfied by latency
	// alone. That leniency exists for the probe pass above and is wrong here: a
	// full run that returned only a latency block would otherwise replace a good
	// record with zeroes, and the card would render dashes under a fresh
	// "Measured just now" -- a worse outcome than keeping the older figures.
	if !ok || result.Down <= 0 || result.Up <= 0 {
		return
	}

	// run_text() flattens a timeout, a non-zero exit and a missing binary into
	// the same empty string, so this branch is also where "cfspeedtest is not
	// installed" lands. Dropping the result is the right answer for all of
	// them: the card returns to whatever it showed before, which is either a
	// real older record or the placeholder.
	end := CollectNetworkIdentity()
	if end.UUID == "" || end.UUID != start.UUID {
		return
	}

	speedtestLock.Lock()
	speedtestRecords[end.UUID] = SpeedtestRecord{
		Name:       end.Name,
		Down:       result.Down,
		Up:         result.Up,
		Latency:    result.Latency,
		MeasuredAt: nowOr(0),
	}
	speedtestRecords = PruneSpeedtestRecords(speedtestRecords, speedtestRecordLimit)
	speedtestLock.Unlock()
}
