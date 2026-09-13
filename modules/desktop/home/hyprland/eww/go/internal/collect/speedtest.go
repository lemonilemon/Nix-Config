package collect

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

// The speed test behind the network popup's card.
//
// cfspeedtest rather than either speedtest.net client: both speedtest-cli and
// speedtest-go fail before transferring a byte, because speedtest-config.php now
// answers an unauthenticated GET with a Cloudflare bot challenge. Ookla's own
// client costs an unfree licence and a local build with no binary cache.
//
// The caveat the number on screen cannot carry: this measures the path to the
// nearest Cloudflare edge, which is peered far better than an average route.
//
// Two invocations rather than one. The probe pass costs 0.7 s and yields the
// round-trip figure; the throughput pass costs 8.7 s and 67.8 MiB. A single full
// run would leave all three fields on placeholders for the whole wait.

const (
	// The light profile. cfspeedtest's defaults come to roughly 361 MB per
	// direction; these settle at 33.8 MiB down and 34.0 MiB up in 8.7 s. On a
	// capped line that difference is the whole decision.
	speedtestRuns        = "3"
	speedtestPayloadCap  = "10m"
	speedtestLatencyRuns = "10"

	// The probe pass: latency plus the smallest payload cfspeedtest offers. The
	// download is only there to keep the invocation well-formed.
	speedtestProbeRuns    = "1"
	speedtestProbePayload = "100k"

	// Generous because the payloads are fixed and the link is not. cfspeedtest
	// skips the larger payload sizes once one exceeds 5 s, which bounds the real
	// worst case well inside this.
	speedtestProbeTimeout = 20 * time.Second
	speedtestTimeout      = 90 * time.Second

	// Rounded up from the measured 67.8 MiB, and stated on the card BEFORE the run.
	speedtestDataNote = "about 70 MB"

	// U+2014 em dash, as an escape: a literal does not survive review.
	speedtestDash = "\u2014"

	speedtestNoRecord = "Not measured on this network"
)

// Speedtest is the popup card's readout. Every field is a rendered string rather
// than a number: eww.yuck has no formatting of its own, so a float reaching it
// would render as Go's default and could not be made to say "86.0".
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
// Deliberately not the whole document: the report also carries the machine's
// public IP address, which must not reach a file on disk or the state blob.
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

// SpeedtestFromJSON reads a cfspeedtest report. The LARGEST payload wins for each
// direction rather than the mean: small transfers are dominated by connection
// setup, and averaging measured 14.9 Mb/s against 85.8 Mb/s on one link.
//
// ok=false means the report held nothing worth showing. Latency with no speed
// measurements is the probe pass, and is ok.
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

// FormatMbps renders for the card's 15px slot: one decimal below 100 and none
// above, because the slot fits "86.0" and not "1024.3".
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
// The boundaries are calendar days, not elapsed hours: a measurement taken at 23:50
// is "yesterday" at 00:10, because that is what a person means by it. A record
// dated in the future falls through to the bare date rather than "-2 days ago".
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

// SpeedtestView is everything the card builder needs about one moment: the stored
// record, whatever a run in flight has produced, and the two overriding states.
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
// The rule it exists to enforce: a figure appears only when it was measured on the
// connection attached right now. A record filed under another UUID renders as
// placeholders, NOT as that network's numbers with a caveat beside them.
//
// Precedence: a run in flight beats everything; no link beats a stored record,
// because a number beside "Disconnected" is a contradiction; a stored record beats
// the portal notice, which the header and footer already announce.
func SpeedtestCardFrom(view SpeedtestView, nowEpoch float64) Speedtest {
	card := SpeedtestDefault()

	if view.Running {
		card.Status = "running"
		card.Caption = "Testing\u2026 uses " + speedtestDataNote
		// The probe pass lands about eight seconds before the throughput pass, so
		// this field is populated for most of the wait.
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

// The in-memory half of the record store. Records mirrors what is on disk;
// running and partial last one run and are never persisted.
var (
	speedtestLock    sync.Mutex
	speedtestRecords = map[string]SpeedtestRecord{}
	speedtestRunning string
	speedtestPartial SpeedtestResult
)

// SeedSpeedtestRecords only fills an EMPTY cache, matching SeedQuotaCache: a
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

// SpeedtestCardFor assembles the card for one connection. An empty uuid can never
// match a record, which is right: with no identity there is no way to know whose
// numbers a stored record holds.
func SpeedtestCardFor(uuid, class, connectivity string, nowEpoch float64) Speedtest {
	speedtestLock.Lock()
	view := SpeedtestView{
		// Compared against this uuid, not merely non-empty. Otherwise a run started
		// on one network reports "testing" on whatever you switch to.
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

// RunSpeedtest files its measurement against the network attached when it FINISHES.
// The run takes about nine seconds, long enough to roam onto a different profile,
// and filing a hotspot's numbers under the cafe's UUID would poison that record
// permanently with nothing on screen to suggest it.
//
// publish is called on every state change the card can show.
func RunSpeedtest(publish func()) {
	start := CollectNetworkIdentity()
	// Refused rather than attempted: without an identity the result has nowhere
	// to be filed, so the run would move 70 MB and discard the answer.
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
	// alone. A full run returning only a latency block would otherwise replace a
	// good record with zeroes under a fresh "Measured just now".
	if !ok || result.Down <= 0 || result.Up <= 0 {
		return
	}

	// run_text() flattens a timeout, a non-zero exit and a missing binary into
	// the same empty string, so "cfspeedtest is not installed" also lands here.
	// Dropping the result is right for all of them.
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
