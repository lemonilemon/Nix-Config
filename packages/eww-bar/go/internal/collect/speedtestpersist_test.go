package collect

import (
	"os"
	"path/filepath"
	"testing"
)

func recordAt(name string, epoch float64) SpeedtestRecord {
	return SpeedtestRecord{Name: name, Down: 86, Up: 58, Latency: 42, MeasuredAt: epoch}
}

func TestSpeedtestSnapshotRoundTrips(t *testing.T) {
	records := map[string]SpeedtestRecord{"uuid-a": recordAt("Tenda_5G", savedEpoch)}

	text, err := EncodeSpeedtestSnapshot(records)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	restored, ok := ParseSpeedtestSnapshot(text)
	if !ok {
		t.Fatal("a snapshot of one record did not restore")
	}
	if restored["uuid-a"] != records["uuid-a"] {
		t.Errorf("restored %+v, want %+v", restored["uuid-a"], records["uuid-a"])
	}
}

// A record that cannot say when it was taken would render as "Measured Jan 1"
// in 1970. The card's contract is that every number on it can be dated, so a
// record that breaks it is dropped rather than shown.
func TestSpeedtestSnapshotDropsUndatedRecords(t *testing.T) {
	restored, ok := ParseSpeedtestSnapshot(
		`{"version":1,"records":{"uuid-a":{"name":"X","down_mbps":86},` +
			`"uuid-b":{"name":"Y","down_mbps":50,"measured_at":1786804320}}}`)
	if !ok {
		t.Fatal("a snapshot with one good record did not restore")
	}
	if _, present := restored["uuid-a"]; present {
		t.Error("an undated record was restored")
	}
	if _, present := restored["uuid-b"]; !present {
		t.Error("the dated record was dropped along with it")
	}
}

// A file written by a newer daemon is not readable by an older one, and
// guessing at it would be worse than starting empty.
func TestSpeedtestSnapshotRefusesAFutureVersion(t *testing.T) {
	if _, ok := ParseSpeedtestSnapshot(
		`{"version":99,"records":{"uuid-a":{"measured_at":1786804320}}}`); ok {
		t.Error("a future version parsed")
	}
}

func TestSpeedtestSnapshotRejectsRubbish(t *testing.T) {
	for _, text := range []string{"", "{", "{}", `{"version":1,"records":{}}`} {
		if _, ok := ParseSpeedtestSnapshot(text); ok {
			t.Errorf("%q parsed as a usable snapshot", text)
		}
	}
}

// A laptop that joins a new network every week would grow this file forever.
// Pruning keeps the networks most recently measured, which are the ones most
// likely to be revisited.
func TestPruneKeepsTheNewestRecords(t *testing.T) {
	records := map[string]SpeedtestRecord{
		"old":    recordAt("Old", savedEpoch-1000),
		"newer":  recordAt("Newer", savedEpoch),
		"newest": recordAt("Newest", savedEpoch+1000),
	}

	kept := PruneSpeedtestRecords(records, 2)
	if len(kept) != 2 {
		t.Fatalf("kept %d records, want 2", len(kept))
	}
	if _, present := kept["old"]; present {
		t.Error("pruning kept the oldest record")
	}
	if _, present := kept["newest"]; !present {
		t.Error("pruning dropped the newest record")
	}
}

// Two writes of the same state must produce the same file, or the atomic write
// churns for nothing and any future golden test is unreproducible.
func TestPruneIsDeterministicOnTies(t *testing.T) {
	records := map[string]SpeedtestRecord{
		"uuid-a": recordAt("A", savedEpoch),
		"uuid-b": recordAt("B", savedEpoch),
		"uuid-c": recordAt("C", savedEpoch),
	}
	first, err := EncodeSpeedtestSnapshot(PruneSpeedtestRecords(records, 2))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	for i := 0; i < 8; i++ {
		again, err := EncodeSpeedtestSnapshot(PruneSpeedtestRecords(records, 2))
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		if again != first {
			t.Fatalf("pruning a tie is not deterministic:\n%s\n%s", first, again)
		}
	}
}

// The only way to reach a save with nothing cached is a save before anything
// was ever measured. Writing there would replace every remembered network with
// an empty file -- the same trade SaveHistory refuses to make.
func TestSaveNeverTruncatesToNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "speedtest.json")
	existing := `{"version":1,"records":{"uuid-a":{"measured_at":1786804320}}}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	speedtestLock.Lock()
	saved := speedtestRecords
	speedtestRecords = map[string]SpeedtestRecord{}
	speedtestLock.Unlock()
	t.Cleanup(func() {
		speedtestLock.Lock()
		speedtestRecords = saved
		speedtestLock.Unlock()
	})

	if err := SaveSpeedtestRecords(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(after) != existing {
		t.Errorf("an empty cache overwrote the stored records:\n%s", after)
	}
}

// StateFile returns "" when there is no home directory to anchor to, and every
// caller reads that as "no persistence available".
func TestSaveWithNoPathIsNotAnError(t *testing.T) {
	if err := SaveSpeedtestRecords(""); err != nil {
		t.Errorf("saving to no path errored: %v", err)
	}
	if _, ok := LoadSpeedtestSnapshot(""); ok {
		t.Error("loading from no path reported success")
	}
}

// Seeding must never overwrite a reading that already won the race, matching
// SeedQuotaCache. At startup this runs beside collectors that may already have
// written something better.
func TestSeedOnlyFillsAnEmptyCache(t *testing.T) {
	speedtestLock.Lock()
	saved := speedtestRecords
	speedtestRecords = map[string]SpeedtestRecord{"live": recordAt("Live", savedEpoch)}
	speedtestLock.Unlock()
	t.Cleanup(func() {
		speedtestLock.Lock()
		speedtestRecords = saved
		speedtestLock.Unlock()
	})

	SeedSpeedtestRecords(map[string]SpeedtestRecord{"restored": recordAt("Old", 1)})

	got := SpeedtestRecordsSnapshot()
	if _, present := got["restored"]; present {
		t.Error("a restore overwrote a cache that already held a live reading")
	}
	if _, present := got["live"]; !present {
		t.Error("the live reading was lost")
	}
}

// A file holding a valid snapshot followed by anything else is not a file this
// wrote, and accepting it on the strength of its first document alone would
// surface whatever a half-restored backup happened to contain.
func TestSnapshotRejectsTrailingContent(t *testing.T) {
	good := `{"version":1,"records":{"uuid-a":{"measured_at":1786804320}}}`
	if _, ok := ParseSpeedtestSnapshot(good); !ok {
		t.Fatal("the control case did not parse")
	}
	if _, ok := ParseSpeedtestSnapshot(good + "\n{\"version\":1}"); ok {
		t.Error("a snapshot with a second document after it was accepted")
	}
	if _, ok := ParseSpeedtestSnapshot(good + " garbage"); ok {
		t.Error("a snapshot with trailing garbage was accepted")
	}
}
