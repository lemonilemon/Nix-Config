package collect

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

// Live throughput on whichever interface owns the default route.
//
// Answers a different question from the speed test beside it: that says what this
// network COULD do, measured once and remembered, while this says what is being
// used right now.
//
// Two file reads and no forks -- /proc/net/route and /proc/net/dev -- which is why
// this can run at 2 s where the nmcli-driven network state runs at 60.
//
// Megabits per second, not megabytes, so the reading can be compared directly
// against the speed card's ceiling two rows down.

// NetRate is the live throughput readout. Device is carried so the popup can say
// which interface the numbers describe: a VPN tunnel taking the default route reads
// very differently from the Wi-Fi underneath it.
type NetRate struct {
	Down   string `json:"down"`
	Up     string `json:"up"`
	Device string `json:"device"`
}

// NetRateDefault is the readout before two samples exist to compare.
//
// Keep it in step with eww.yuck's net_rate defvar. Nothing in the build checks
// that -- check_yuck_initial.py reads only line 1 of the file -- so the same
// caveat applies as to ai_history:
//
//	go run ./internal/collect/diffgen <<< '{"fn":"NetRateDefault","args":[]}'
func NetRateDefault() NetRate {
	return NetRate{Down: speedtestDash, Up: speedtestDash, Device: ""}
}

// DefaultRouteInterfaceFromProc picks the interface owning the default route out of
// /proc/net/route: zero destination AND zero mask, lowest metric winning. Reads the
// kernel's table directly rather than forking `ip` at this tick rate.
func DefaultRouteInterfaceFromProc(routeText string) string {
	best, bestMetric, found := "", 0, false
	for index, line := range SplitLines(routeText) {
		// The first line is the column header, which has "Destination" where a
		// route has 00000000 and would otherwise parse as a malformed entry.
		if index == 0 {
			continue
		}
		fields := SplitWhitespaceN(line, -1)
		// Destination AND mask must both be zero. A split-default VPN route of
		// 0.0.0.0/1 also has a zero destination but a non-zero mask, and matching on
		// destination alone would meter the tunnel while claiming to meter the link.
		if len(fields) < 8 || fields[1] != "00000000" || fields[7] != "00000000" {
			continue
		}
		// RTF_UP. A route the kernel is not currently using carries no traffic,
		// so metering its interface would report a flat zero.
		if flags, err := strconv.ParseUint(fields[3], 16, 32); err != nil || flags&0x1 == 0 {
			continue
		}
		metric, err := strconv.Atoi(fields[6])
		if err != nil {
			continue
		}
		if !found || metric < bestMetric {
			best, bestMetric, found = fields[0], metric, true
		}
	}
	return best
}

// InterfaceBytesFromProc reads one interface's cumulative counters out of
// /proc/net/dev. Lines look like:
//
//	wlo1: 1234567  890 0 0 0 0 0 0  76543  210 0 0 0 0 0 0
//
// Split on the colon, not on whitespace: a long interface name runs right up
// against it, so "enp0s31f6:1234" is one field and the counters shift a column.
func InterfaceBytesFromProc(devText, device string) (uint64, uint64, bool) {
	for _, line := range SplitLines(devText) {
		name, rest, found := strings.Cut(line, ":")
		if !found || Strip(name) != device {
			continue
		}
		fields := SplitWhitespaceN(rest, -1)
		// 8 receive columns then 8 transmit; bytes is the first of each.
		if len(fields) < 9 {
			return 0, 0, false
		}
		received, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return 0, 0, false
		}
		transmitted, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			return 0, 0, false
		}
		return received, transmitted, true
	}
	return 0, 0, false
}

// FormatRateMbps renders bytes per second as megabits per second. Zero renders as
// "0.0" rather than the speed card's placeholder: a card with no record has never
// been measured, while a rate of zero IS a measurement.
func FormatRateMbps(bytesPerSecond float64) string {
	if bytesPerSecond < 0 {
		return speedtestDash
	}
	megabits := bytesPerSecond * 8 / 1e6
	if megabits >= 100 {
		return strconv.Itoa(RoundHalfEven(megabits))
	}
	return strconv.FormatFloat(megabits, 'f', 1, 64)
}

// netRateSample is the previous reading a rate is measured against.
type netRateSample struct {
	device      string
	received    uint64
	transmitted uint64
	at          time.Time
	valid       bool
}

var (
	netRateLock sync.Mutex
	netRateLast netRateSample
)

// ResetNetRate drops the previous sample. Exported for tests; nothing in
// production calls it, because every reason to discard a sample is detected
// inside RateFromSamples.
func ResetNetRate() {
	netRateLock.Lock()
	defer netRateLock.Unlock()
	netRateLast = netRateSample{}
}

// RateFromSamples turns two counter readings into a rate, reporting ok=false when
// the pair cannot be trusted: no previous sample, a different interface, counters
// that went backwards (an interface taken down and brought back up), or no elapsed
// time. All four mean the same thing to the caller -- show placeholders and wait.
func RateFromSamples(previous, current netRateSample) (float64, float64, bool) {
	if !previous.valid || previous.device != current.device {
		return 0, 0, false
	}
	if current.received < previous.received || current.transmitted < previous.transmitted {
		return 0, 0, false
	}
	elapsed := current.at.Sub(previous.at).Seconds()
	if elapsed <= 0 {
		return 0, 0, false
	}
	return float64(current.received-previous.received) / elapsed,
		float64(current.transmitted-previous.transmitted) / elapsed,
		true
}

// CollectNetRate samples the counters and reports the rate since the last call.
func CollectNetRate() NetRate {
	routeText, _ := ReadTextFile("/proc/net/route")
	device := DefaultRouteInterfaceFromProc(routeText)
	if device == "" {
		ResetNetRate()
		return NetRateDefault()
	}

	devText, _ := ReadTextFile("/proc/net/dev")
	received, transmitted, ok := InterfaceBytesFromProc(devText, device)
	if !ok {
		ResetNetRate()
		return NetRateDefault()
	}

	current := netRateSample{
		device: device, received: received, transmitted: transmitted,
		at: timeNow(), valid: true,
	}

	netRateLock.Lock()
	down, up, usable := RateFromSamples(netRateLast, current)
	netRateLast = current
	netRateLock.Unlock()

	if !usable {
		rate := NetRateDefault()
		rate.Device = device
		return rate
	}
	return NetRate{
		Down:   FormatRateMbps(down),
		Up:     FormatRateMbps(up),
		Device: device,
	}
}
