package collect

import (
	"strings"
	"testing"
)

// A cfspeedtest report shaped like a real one, trimmed to the fields read. The
// three payload sizes and their spread are the point: the 100 kB pass measured
// 14.9 Mb/s and the 10 MB pass 85.8 Mb/s on the same link.
const cfReport = `{
  "metadata": {"country": "TW", "ip": "203.0.113.7", "colo": "TPE"},
  "latency_measurement": {"avg_latency_ms": 42.059, "min_latency_ms": 32.48},
  "speed_measurements": [
    {"test_type": "Download", "payload_size": 100000,   "avg": 14.928},
    {"test_type": "Download", "payload_size": 1000000,  "avg": 61.562},
    {"test_type": "Download", "payload_size": 10000000, "avg": 85.813},
    {"test_type": "Upload",   "payload_size": 100000,   "avg": 18.101},
    {"test_type": "Upload",   "payload_size": 1000000,  "avg": 50.753},
    {"test_type": "Upload",   "payload_size": 10000000, "avg": 58.234}
  ]
}`

// The probe pass: latency, and one token download to keep the invocation
// well-formed.
const cfProbeReport = `{
  "latency_measurement": {"avg_latency_ms": 57.151},
  "speed_measurements": [
    {"test_type": "Download", "payload_size": 100000, "avg": 9.909}
  ]
}`

// Averaging the payload sizes together would report about a third of the real
// throughput, because the small transfers are mostly connection setup.
func TestSpeedtestReadsTheLargestPayload(t *testing.T) {
	result, ok := SpeedtestFromJSON(cfReport)
	if !ok {
		t.Fatal("a complete report did not parse")
	}
	if result.Down != 85.813 {
		t.Errorf("download is %v, want the 10 MB pass at 85.813", result.Down)
	}
	if result.Up != 58.234 {
		t.Errorf("upload is %v, want the 10 MB pass at 58.234", result.Up)
	}
	if result.Latency != 42.059 {
		t.Errorf("latency is %v, want 42.059", result.Latency)
	}
}

// The probe exists to put a round-trip figure on the card about eight seconds
// before the throughput pass finishes. A parser that demanded both directions
// would discard it and the split would buy nothing.
func TestSpeedtestAcceptsALatencyOnlyProbe(t *testing.T) {
	result, ok := SpeedtestFromJSON(cfProbeReport)
	if !ok {
		t.Fatal("the probe pass did not parse")
	}
	if result.Latency != 57.151 {
		t.Errorf("latency is %v, want 57.151", result.Latency)
	}
	if result.Up != 0 {
		t.Errorf("upload is %v, want zero -- the probe does not measure it", result.Up)
	}
}

// run_text() flattens a timeout, a non-zero exit and a missing binary into the
// same empty string, so this is the branch every operational failure lands in.
// It has to be distinguishable from a real reading of zero.
func TestSpeedtestRejectsNothing(t *testing.T) {
	for _, text := range []string{"", "not json", "{}", `{"speed_measurements": []}`} {
		if _, ok := SpeedtestFromJSON(text); ok {
			t.Errorf("%q parsed as a usable result", text)
		}
	}
}

func TestFormatMbpsFitsTheCard(t *testing.T) {
	for _, c := range []struct {
		value float64
		want  string
	}{
		{86.02, "86.0"},
		{9.909, "9.9"},
		{100, "100"},
		{940.7, "941"},
		{0, "—"},
		{-1, "—"},
	} {
		if got := FormatMbps(c.value); got != c.want {
			t.Errorf("FormatMbps(%v) = %q, want %q", c.value, got, c.want)
		}
	}
}

func TestFormatLatencyIsWholeMilliseconds(t *testing.T) {
	if got := FormatLatency(42.059); got != "42" {
		t.Errorf("FormatLatency(42.059) = %q, want %q", got, "42")
	}
	if got := FormatLatency(0); got != "—" {
		t.Errorf("FormatLatency(0) = %q, want a placeholder", got)
	}
}

// The age label is the whole reason a stored number is safe to show. Without
// it the card would present a measurement from March as though it had just
// run, and nothing on screen would contradict it.
func TestSpeedtestAgeSaysHowOldTheNumberIs(t *testing.T) {
	pinClock(t, sameDayLaterEpoch)

	for _, c := range []struct {
		name    string
		epoch   float64
		want    string
		wantSub string
	}{
		{name: "same day keeps the minute", epoch: savedEpoch, want: "Measured 14:32"},
		{name: "yesterday is named", epoch: savedEpoch - 86400, want: "Measured yesterday"},
		{name: "this week counts days", epoch: savedEpoch - 3*86400, want: "Measured 3 days ago"},
		{name: "older falls back to a date", epoch: savedEpoch - 30*86400, wantSub: "Measured Jul"},
		// Clock skew, or a record restored from a machine that disagrees about
		// the time. "Measured -2 days ago" would be worse than a date.
		{name: "the future falls back to a date", epoch: savedEpoch + 5*86400, wantSub: "Measured Aug"},
	} {
		got := SpeedtestAge(c.epoch, 0)
		if c.want != "" && got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if c.wantSub != "" && !strings.HasPrefix(got, c.wantSub) {
			t.Errorf("%s: got %q, want a date starting %q", c.name, got, c.wantSub)
		}
	}
}

func aRecord() SpeedtestRecord {
	return SpeedtestRecord{
		Name: "Tenda_5G", Down: 85.813, Up: 58.234,
		Latency: 42.059, MeasuredAt: savedEpoch,
	}
}

// The rule the whole feature exists to enforce.
func TestCardShowsNumbersOnlyForTheAttachedNetwork(t *testing.T) {
	pinClock(t, sameDayLaterEpoch)

	card := SpeedtestCardFrom(SpeedtestView{
		Record: aRecord(), HaveRecord: true, Class: "wifi", Connectivity: "full",
	}, 0)
	if card.Down != "85.8" || card.Status != "ready" {
		t.Fatalf("a record for this network did not render: %+v", card)
	}

	// The same moment, on a network with no record of its own. The previous
	// network's 85.8 must not follow the user here, with or without a caveat.
	elsewhere := SpeedtestCardFrom(SpeedtestView{
		HaveRecord: false, Class: "wifi", Connectivity: "full",
	}, 0)
	if elsewhere.Down != "—" || elsewhere.Up != "—" || elsewhere.Latency != "—" {
		t.Errorf("another network's numbers surfaced: %+v", elsewhere)
	}
	if elsewhere.Status != "none" {
		t.Errorf("status is %q, want \"none\"", elsewhere.Status)
	}
	if strings.Contains(elsewhere.Caption, "85.8") {
		t.Errorf("caption leaks the other network's figure: %q", elsewhere.Caption)
	}
}

// A run in flight outranks everything, because it is the one state the reader
// caused and the only one where nothing on the card is trustworthy yet.
func TestRunningCardShowsTheProbeLatencyAndTheDataCost(t *testing.T) {
	card := SpeedtestCardFrom(SpeedtestView{
		Record: aRecord(), HaveRecord: true, Running: true,
		Partial: SpeedtestResult{Latency: 38.4},
		Class:   "wifi", Connectivity: "full",
	}, 0)

	if card.Status != "running" {
		t.Fatalf("status is %q, want \"running\"", card.Status)
	}
	if card.Latency != "38" {
		t.Errorf("latency is %q, want the probe's 38 -- the split run exists "+
			"so this lands before the throughput pass", card.Latency)
	}
	if card.Down != "—" {
		t.Errorf("download is %q, want a placeholder: the stale figure must "+
			"not sit beside a live latency as though both were current", card.Down)
	}
	// The cost is stated before the user is committed, not after.
	if !strings.Contains(card.Caption, "70 MB") {
		t.Errorf("caption %q does not say what the run costs", card.Caption)
	}
}

// A number beside "Disconnected" is a contradiction, so no link outranks a
// stored record.
func TestDisconnectedCardShowsNoNumbers(t *testing.T) {
	card := SpeedtestCardFrom(SpeedtestView{
		Record: aRecord(), HaveRecord: true, Class: "disconnected",
	}, 0)
	if card.Status != "offline" || card.Down != "—" {
		t.Errorf("a disconnected card kept its numbers: %+v", card)
	}
}

// Behind a portal with nothing measured, the card says what to do about it.
// With a record, the record wins: having one proves the portal was signed into
// before, and the header and footer already announce the portal loudly.
func TestPortalCardDefersToARecord(t *testing.T) {
	pinClock(t, sameDayLaterEpoch)

	blocked := SpeedtestCardFrom(SpeedtestView{
		Class: "wifi", Connectivity: "portal",
	}, 0)
	if blocked.Status != "blocked" || !strings.Contains(blocked.Caption, "Sign in") {
		t.Errorf("a portal with no record did not say to sign in: %+v", blocked)
	}

	known := SpeedtestCardFrom(SpeedtestView{
		Record: aRecord(), HaveRecord: true, Class: "wifi", Connectivity: "portal",
	}, 0)
	if known.Status != "ready" || known.Down != "85.8" {
		t.Errorf("a portal hid a record it should have shown: %+v", known)
	}
}

// With no identity there is no way to know whose numbers a record holds, so an
// empty UUID must never match one.
func TestCardForRefusesToMatchAnEmptyUUID(t *testing.T) {
	pinClock(t, sameDayLaterEpoch)

	speedtestLock.Lock()
	saved := speedtestRecords
	speedtestRecords = map[string]SpeedtestRecord{"": aRecord(), "uuid-a": aRecord()}
	speedtestLock.Unlock()
	t.Cleanup(func() {
		speedtestLock.Lock()
		speedtestRecords = saved
		speedtestLock.Unlock()
	})

	if card := SpeedtestCardFor("", "wifi", "full", 0); card.Status != "none" {
		t.Errorf("an empty UUID matched a record: %+v", card)
	}
	if card := SpeedtestCardFor("uuid-a", "wifi", "full", 0); card.Status != "ready" {
		t.Errorf("a known UUID did not match its record: %+v", card)
	}
	if card := SpeedtestCardFor("uuid-b", "wifi", "full", 0); card.Status != "none" {
		t.Errorf("an unknown UUID matched some other network's record: %+v", card)
	}
}

// "Running" belongs to the network under test, not to whoever asks. A run
// started on one network used to report "testing" on whatever you switched to,
// and showed that run's latency beside the new network's name.
func TestRunningStateBelongsToTheNetworkUnderTest(t *testing.T) {
	pinClock(t, sameDayLaterEpoch)

	speedtestLock.Lock()
	savedRunning, savedPartial := speedtestRunning, speedtestPartial
	speedtestRunning = "uuid-a"
	speedtestPartial = SpeedtestResult{Latency: 38}
	speedtestLock.Unlock()
	t.Cleanup(func() {
		speedtestLock.Lock()
		speedtestRunning, speedtestPartial = savedRunning, savedPartial
		speedtestLock.Unlock()
	})

	if card := SpeedtestCardFor("uuid-a", "wifi", "full", 0); card.Status != "running" {
		t.Errorf("the network under test reports %q", card.Status)
	}
	other := SpeedtestCardFor("uuid-b", "wifi", "full", 0)
	if other.Status == "running" {
		t.Error("a different network reports a run it is not part of")
	}
	if other.Latency == "38" {
		t.Error("a different network shows the latency measured on uuid-a")
	}
}
