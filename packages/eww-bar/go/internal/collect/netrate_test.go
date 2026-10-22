package collect

import (
	"testing"
	"time"
)

// Real /proc/net/route, header included. Destination 00000000 is the default;
// the second line is the on-link subnet and must not be mistaken for one.
const procRoute = "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n" +
	"wlo1\t00000000\t0100A8C0\t0003\t0\t0\t600\t00000000\t0\t0\t0\n" +
	"wlo1\t0000A8C0\t00000000\t0001\t0\t0\t600\t00FFFFFF\t0\t0\t0\n"

func TestDefaultRouteInterfaceSkipsTheHeaderAndSubnetRoutes(t *testing.T) {
	if got := DefaultRouteInterfaceFromProc(procRoute); got != "wlo1" {
		t.Errorf("got %q, want wlo1", got)
	}
}

// A docked laptop carries a default route per link and the kernel prefers the
// lowest metric. Reading the wrong one would meter an idle interface and report
// zero while the real link was saturated.
func TestDefaultRouteInterfacePrefersTheLowestMetric(t *testing.T) {
	text := "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n" +
		"wlo1\t00000000\t0100A8C0\t0003\t0\t0\t600\t00000000\t0\t0\t0\n" +
		"eno1\t00000000\t0100A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n"
	if got := DefaultRouteInterfaceFromProc(text); got != "eno1" {
		t.Errorf("got %q, want eno1 -- metric 100 beats 600", got)
	}
}

func TestDefaultRouteInterfaceHandlesNoDefaultRoute(t *testing.T) {
	for _, text := range []string{"", "garbage", procRouteHeaderOnly} {
		if got := DefaultRouteInterfaceFromProc(text); got != "" {
			t.Errorf("%q produced interface %q", text, got)
		}
	}
}

const procRouteHeaderOnly = "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n"

const procDev = "Inter-|   Receive                    |  Transmit\n" +
	" face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\n" +
	"    lo:  126257    1368    0    0    0     0          0         0   126257    1368    0    0    0     0       0          0\n" +
	"  wlo1: 987654321  12345    0    0    0     0          0         0  123456789   6789    0    0    0     0       0          0\n"

func TestInterfaceBytesReadsBothDirections(t *testing.T) {
	received, transmitted, ok := InterfaceBytesFromProc(procDev, "wlo1")
	if !ok {
		t.Fatal("wlo1 was not found")
	}
	if received != 987654321 {
		t.Errorf("received %d, want 987654321", received)
	}
	if transmitted != 123456789 {
		t.Errorf("transmitted %d, want 123456789 -- the 9th column, not the 2nd", transmitted)
	}
}

// A long interface name runs straight into the colon with no space, so the line
// splits into one fewer whitespace field and every counter shifts a column. The
// symptom would be a plausible-looking rate that is simply the wrong number.
func TestInterfaceBytesHandlesNamesTouchingTheColon(t *testing.T) {
	text := "  enp0s31f6:1234567 100 0 0 0 0 0 0 7654321 200 0 0 0 0 0 0\n"
	received, transmitted, ok := InterfaceBytesFromProc(text, "enp0s31f6")
	if !ok {
		t.Fatal("enp0s31f6 was not found")
	}
	if received != 1234567 || transmitted != 7654321 {
		t.Errorf("got %d/%d, want 1234567/7654321", received, transmitted)
	}
}

func TestInterfaceBytesMissingDevice(t *testing.T) {
	if _, _, ok := InterfaceBytesFromProc(procDev, "tun0"); ok {
		t.Error("an absent interface reported success")
	}
}

func sample(device string, rx, tx uint64, at time.Time) netRateSample {
	return netRateSample{device: device, received: rx, transmitted: tx, at: at, valid: true}
}

func TestRateFromSamplesComputesPerSecond(t *testing.T) {
	base := time.Unix(1786972523, 0)
	down, up, ok := RateFromSamples(
		sample("wlo1", 1_000_000, 500_000, base),
		sample("wlo1", 3_000_000, 1_500_000, base.Add(2*time.Second)),
	)
	if !ok {
		t.Fatal("two good samples did not produce a rate")
	}
	if down != 1_000_000 {
		t.Errorf("down %v B/s, want 1000000", down)
	}
	if up != 500_000 {
		t.Errorf("up %v B/s, want 500000", up)
	}
}

// Each of these would otherwise surface as a wrong number rather than as an
// absent one, which is the worse failure for a meter.
func TestRateFromSamplesRejectsUntrustworthyPairs(t *testing.T) {
	base := time.Unix(1786972523, 0)
	later := base.Add(2 * time.Second)
	good := sample("wlo1", 3_000_000, 1_500_000, later)

	for _, c := range []struct {
		name     string
		previous netRateSample
		current  netRateSample
	}{
		{"no previous sample", netRateSample{}, good},
		{
			"interface changed",
			sample("tun0", 1_000_000, 500_000, base), good,
		},
		{
			"counters went backwards",
			sample("wlo1", 9_000_000, 500_000, base), good,
		},
		{
			"no elapsed time",
			sample("wlo1", 1_000_000, 500_000, later), good,
		},
	} {
		if _, _, ok := RateFromSamples(c.previous, c.current); ok {
			t.Errorf("%s: produced a rate anyway", c.name)
		}
	}
}

// Same units as the speed card two rows down, so the reading can be read
// against the ceiling instead of being off by a factor of eight.
func TestFormatRateIsMegabitsPerSecond(t *testing.T) {
	for _, c := range []struct {
		bytesPerSecond float64
		want           string
	}{
		{1_000_000, "8.0"},  // 1 MB/s is 8 Mb/s
		{125_000, "1.0"},    //
		{0, "0.0"},          // a real measurement: nothing is using the link
		{15_000_000, "120"}, // three digits drop the decimal to fit the slot
		{-1, "—"},           // not a rate at all
	} {
		if got := FormatRateMbps(c.bytesPerSecond); got != c.want {
			t.Errorf("FormatRateMbps(%v) = %q, want %q", c.bytesPerSecond, got, c.want)
		}
	}
}

// Zero is a reading, not a gap. The speed card uses the placeholder to mean
// "never measured", and a meter that borrowed it would say the same thing about
// an idle connection it says about a broken collector.
func TestIdleRateIsNotThePlaceholder(t *testing.T) {
	if FormatRateMbps(0) == NetRateDefault().Down {
		t.Error("an idle link renders identically to no data at all")
	}
}

// A split-default VPN route (0.0.0.0/1) has a zero DESTINATION but a non-zero
// mask, and usually a lower metric than the real default. Matching on
// destination alone would meter the tunnel while claiming to meter the link.
func TestDefaultRouteInterfaceIgnoresSplitDefaultRoutes(t *testing.T) {
	text := procRouteHeaderOnly +
		"wlo1\t00000000\t0100A8C0\t0003\t0\t0\t600\t00000000\t0\t0\t0\n" +
		"tun0\t00000000\t00000000\t0003\t0\t0\t50\t00000080\t0\t0\t0\n"
	if got := DefaultRouteInterfaceFromProc(text); got != "wlo1" {
		t.Errorf("got %q, want wlo1 -- tun0's 0.0.0.0/1 is not a default route", got)
	}
}

// A route the kernel is not using carries nothing, so metering its interface
// would report a flat zero while the real link was busy.
func TestDefaultRouteInterfaceSkipsRoutesThatAreDown(t *testing.T) {
	text := procRouteHeaderOnly +
		"wlo1\t00000000\t0100A8C0\t0003\t0\t0\t600\t00000000\t0\t0\t0\n" +
		"eno1\t00000000\t0100A8C0\t0002\t0\t0\t100\t00000000\t0\t0\t0\n"
	if got := DefaultRouteInterfaceFromProc(text); got != "wlo1" {
		t.Errorf("got %q, want wlo1 -- eno1's route lacks RTF_UP", got)
	}
}
