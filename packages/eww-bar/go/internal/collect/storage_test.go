package collect

import (
	"strings"
	"testing"
	"time"
)

// 2026-10-22 12:00 UTC, a Thursday.
const storageNow = 1792670400.0

func inUTC(t *testing.T) {
	t.Helper()
	savedZone := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = savedZone })
}

func TestFormatSizeSteps(t *testing.T) {
	cases := map[float64]string{
		512 << 20:        "512M",
		7.44 * (1 << 30): "7.4G",
		17.2 * (1 << 30): "17G",
		369 * (1 << 30):  "369G",
		1500 * (1 << 30): "1.5T",
	}
	for bytes, want := range cases {
		if got := FormatSize(bytes); got != want {
			t.Errorf("FormatSize(%.0f) = %s, want %s", bytes, got, want)
		}
	}
}

// Real output of `systemctl show a.service a.timer --timestamp=unix -p ...`:
// systemd's own property order, a blank line between units, empty stamps for
// never.
func cleanupShow(serviceActive, result, serviceStart, lastTrigger, timerActive, next string) string {
	return "LoadState=loaded\nActiveState=" + serviceActive + "\nResult=" + result +
		"\nExecMainStartTimestamp=" + serviceStart + "\n\n" +
		"LoadState=loaded\nActiveState=" + timerActive + "\nNextElapseUSecRealtime=" + next +
		"\nLastTriggerUSec=" + lastTrigger + "\nResult=success\n"
}

func TestParseUnitShowSplitsUnitsInArgumentOrder(t *testing.T) {
	units := ParseUnitShow(cleanupShow("inactive", "success", "", "@1792500000", "active", "@1792800000"))
	if len(units) != 2 {
		t.Fatalf("got %d units, want 2", len(units))
	}
	if units[0]["ActiveState"] != "inactive" || units[1]["NextElapseUSecRealtime"] != "@1792800000" {
		t.Errorf("units parsed as %v", units)
	}
}

func TestCleanupStateReportsAHostWithoutTheTimer(t *testing.T) {
	missing := "LoadState=not-found\nActiveState=inactive\n\nLoadState=not-found\nActiveState=inactive\n"
	got := CleanupStateFromParts(missing, "", storageNow)
	if got.Status != "missing" || got.Next != "" {
		t.Errorf("got %+v, want missing with no next line", got)
	}
}

func TestCleanupStateNeverRun(t *testing.T) {
	inUTC(t)
	got := CleanupStateFromParts(cleanupShow("inactive", "success", "", "", "active", "@1792972800"), "", storageNow)
	want := StorageCleanup{Status: "idle", Last: "Never", Next: "Monday"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// After a reboot the service has forgotten its last start; the persistent
// timer's trigger is what still knows, and the journal supplies the figure.
func TestCleanupStateUsesTheTimerAndTheJournalAfterAReboot(t *testing.T) {
	inUTC(t)
	journal := "Welcome to nh clean\n1012 store paths deleted, 12.40 GiB freed\n"
	got := CleanupStateFromParts(
		cleanupShow("inactive", "success", "", "@1792584000", "active", "@1793577600"), journal, storageNow)
	want := StorageCleanup{Status: "idle", Last: "yesterday · freed 12G", Next: "Nov 2"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCleanupStateReportsFailureFromEitherSource(t *testing.T) {
	inUTC(t)
	fromService := CleanupStateFromParts(
		cleanupShow("failed", "exit-code", "@1792663200", "", "active", "@1792972800"), "", storageNow)
	if fromService.Status != "failed" || fromService.Last != "Failed today 10:00" {
		t.Errorf("service result: got %+v", fromService)
	}
	journal := "nh-clean.service: Main process exited, code=exited, status=1/FAILURE\n" +
		"nh-clean.service: Failed with result 'exit-code'.\n"
	fromJournal := CleanupStateFromParts(
		cleanupShow("inactive", "success", "", "@1792411200", "active", "@1792972800"), journal, storageNow)
	if fromJournal.Status != "failed" || fromJournal.Last != "Failed 3 days ago" {
		t.Errorf("journal: got %+v", fromJournal)
	}
}

func TestCleanupStateWhileRunningAndWithTheTimerStopped(t *testing.T) {
	got := CleanupStateFromParts(
		cleanupShow("activating", "success", "@1792670000", "@1792670000", "inactive", ""), "", storageNow)
	want := StorageCleanup{Status: "running", Last: "Running now", Next: "Timer stopped"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCleanupJournalReadsTheLastFreedLine(t *testing.T) {
	freed, failed := CleanupJournal("3 store paths deleted, 0.0 KiB freed\n9 store paths deleted, 512.00 MiB freed\n")
	if freed != "512M" || failed {
		t.Errorf("got %q %v, want 512M without failure", freed, failed)
	}
}

// 352 GiB used and 17 GiB available is 95.4%; df rounds that up to 96.
func TestStorageStateMatchesDfAndFlagsAFullDisk(t *testing.T) {
	const gib = 1 << 30
	fs := FsUsage{Total: 388 * gib, Free: 36 * gib, Avail: 17 * gib, OK: true}
	got := StorageStateFromParts(fs, StorageCleanup{Status: "idle", Last: "Never", Next: "Monday"})
	if got.UsedPercent != 96 || got.Class != "critical" || got.Text != glyphStorage+" 17G" {
		t.Errorf("got %+v", got)
	}
	if got.Used != "352G of 388G" || !strings.Contains(got.Tooltip, "\nLast cleanup: Never\nNext cleanup: Monday") {
		t.Errorf("got used %q tooltip %q", got.Used, got.Tooltip)
	}
}

func TestStorageStateThresholds(t *testing.T) {
	const gib = 1 << 30
	for avail, want := range map[uint64]string{12 * gib: "", 8 * gib: "warning", 4 * gib: "critical"} {
		fs := FsUsage{Total: 100 * gib, Free: avail, Avail: avail, OK: true}
		if got := StorageStateFromParts(fs, StorageCleanup{Status: "missing"}).Class; got != want {
			t.Errorf("%d GiB free: class %q, want %q", avail/gib, got, want)
		}
	}
}

func TestStorageStateWithoutAReadingKeepsTheCleanupLine(t *testing.T) {
	cleanup := StorageCleanup{Status: "missing"}
	got := StorageStateFromParts(FsUsage{}, cleanup)
	if got.Text != StorageDefault().Text || got.Cleanup != cleanup {
		t.Errorf("got %+v", got)
	}
}
