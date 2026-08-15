package collect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// liveCards is a probe result worth remembering: three providers, all live.
func liveCards() []Quota {
	return []Quota{
		{Key: "claude", Name: "Claude", Plan: "Max", Status: "live", Class: "active",
			Updated: "14:32", Windows: []QuotaWindow{
				{Label: "Session", Percent: 41, Value: "41%", Remaining: "2h 10m",
					Reset: "16:42", Class: "active"},
			}, Meta: []QuotaMeta{}},
		{Key: "codex", Name: "Codex", Plan: "Pro", Status: "live", Class: "empty",
			Updated: "14:32", Windows: []QuotaWindow{}, Meta: []QuotaMeta{}},
		{Key: "antigravity", Name: "Antigravity", Plan: "--", Status: "live", Class: "empty",
			Updated: "14:32", Windows: []QuotaWindow{}, Meta: []QuotaMeta{}},
	}
}

func cardByKey(t *testing.T, cards []Quota, key string) Quota {
	t.Helper()
	for _, card := range cards {
		if card.Key == key {
			return card
		}
	}
	t.Fatalf("no card %q in %v", key, cards)
	return Quota{}
}

// 2026-08-15 14:32:00 UTC, read in UTC because pinClock forces that zone.
const (
	savedEpoch          = 1786804320
	sameDayLaterEpoch   = savedEpoch + 3*3600  // 17:32 the same day
	threeDaysLaterEpoch = savedEpoch + 3*86400 // 2026-08-18
)

// pinClock freezes timeNow and the zone for the duration of a test.
//
// The zone is not incidental. restoredClock decides between a bare time and a
// date by comparing LOCAL calendar days, so the same pair of epochs is
// same-day in one zone and consecutive days in another: at UTC+8 the two
// constants above straddle midnight. Package collect has no TestMain, so
// time.Local is whatever the machine says -- CST+0800 on this host, UTC in the
// Nix sandbox, which has no /etc/localtime. Pinning it here is what keeps this
// test from passing locally and failing in the build, the failure mode
// replay_test.go:46 documents having already been through once.
func pinClock(t *testing.T, epoch float64) {
	t.Helper()
	savedNow, savedZone := timeNow, time.Local
	timeNow = func() time.Time { return time.Unix(int64(epoch), 0) }
	time.Local = time.UTC
	t.Cleanup(func() { timeNow, time.Local = savedNow, savedZone })
}

// A card carries Updated as a bare HH:MM with no date. Restoring one unchanged
// would render yesterday's 14:32 as though the probe had just run, and nothing
// on screen would contradict it -- the failure is silent and the user acts on a
// number that is a day old.
func TestRestoredQuotaCardsSayWhenTheyWereTaken(t *testing.T) {
	pinClock(t, sameDayLaterEpoch)

	text, err := EncodeQuotaSnapshot(liveCards(), savedEpoch)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	restored, ok := ParseQuotaSnapshot(text)
	if !ok {
		t.Fatal("a snapshot of three live cards did not restore")
	}

	claude := cardByKey(t, restored, "claude")
	if claude.Status == "live" {
		t.Error("a restored card still claims to be live")
	}
	// eww.yuck renders quota.status verbatim once it is not "live", so this
	// string is what reaches the screen.
	if claude.Status != "as of 14:32" {
		t.Errorf("status = %q, want \"as of 14:32\"", claude.Status)
	}
}

// A week-old snapshot rendered as a bare "as of 14:32" is the exact ambiguity
// this label exists to remove, so an older save has to name the day instead.
func TestRestoredCardsFromAnEarlierDayShowTheDate(t *testing.T) {
	pinClock(t, threeDaysLaterEpoch)

	text, _ := EncodeQuotaSnapshot(liveCards(), savedEpoch)

	restored, ok := ParseQuotaSnapshot(text)
	if !ok {
		t.Fatal("did not restore")
	}

	claude := cardByKey(t, restored, "claude")
	if claude.Status != "as of Aug 15" {
		t.Errorf("status = %q, want \"as of Aug 15\"", claude.Status)
	}
	if strings.Contains(claude.Status, ":") {
		t.Errorf("status = %q, a clock time on a three-day-old snapshot reads as today", claude.Status)
	}
}

// The numbers are the reason to restore at all. A label that says "as of" while
// the bars are blank is no better than the "--" this replaces.
func TestRestoredQuotaCardsKeepTheirWindows(t *testing.T) {
	text, _ := EncodeQuotaSnapshot(liveCards(), savedEpoch)

	restored, ok := ParseQuotaSnapshot(text)
	if !ok {
		t.Fatal("did not restore")
	}

	claude := cardByKey(t, restored, "claude")
	if len(claude.Windows) != 1 {
		t.Fatalf("windows = %v, want the saved one kept", claude.Windows)
	}
	if claude.Windows[0].Percent != 41 {
		t.Errorf("percent = %d, want 41", claude.Windows[0].Percent)
	}
}

// A remembered failure is worse than no memory. "not logged in" restored at
// boot blames the user for a condition that a since-completed login may have
// cleared, and it looks identical to a live reading of the same problem.
func TestRestoreDropsCardsThatWereNotLive(t *testing.T) {
	cards := liveCards()
	cards[1].Status = "not logged in"
	cards[1].Windows = []QuotaWindow{}
	text, _ := EncodeQuotaSnapshot(cards, savedEpoch)

	restored, ok := ParseQuotaSnapshot(text)
	if !ok {
		t.Fatal("one dead card should not sink the whole snapshot")
	}

	codex := cardByKey(t, restored, "codex")
	if codex.Status != "waiting" {
		t.Errorf("status = %q, want the blank template's 'waiting'", codex.Status)
	}
}

// If nothing in the file was ever live there is nothing worth showing, and the
// caller should fall through to the normal defaults.
func TestRestoreRejectsASnapshotWithNoLiveCards(t *testing.T) {
	cards := liveCards()
	for i := range cards {
		cards[i].Status = "unavailable"
	}
	text, _ := EncodeQuotaSnapshot(cards, savedEpoch)

	if _, ok := ParseQuotaSnapshot(text); ok {
		t.Error("an all-dead snapshot restored")
	}
}

func TestParseQuotaSnapshotSurvivesGarbage(t *testing.T) {
	for _, text := range []string{"", "{", "null", `{"version":99,"quotas":[]}`, `{"version":1,"quotas":[]}`} {
		if _, ok := ParseQuotaSnapshot(text); ok {
			t.Errorf("input %q restored", text)
		}
	}
}

// A save that captured one good card and two failures would be restored next
// boot as a card set that never existed at the same moment.
func TestSaveQuotaSnapshotRefusesAPartialSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-quotas.json")
	cards := liveCards()
	cards[2].Status = "unavailable"

	if err := SaveQuotaSnapshot(path, cards, savedEpoch); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a partial card set was written")
	}
}

// A refresh that found nothing must not be the thing that empties the file --
// that is precisely when the stored copy is the only copy.
func TestSaveHistoryNeverTruncatesToEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-history.json")
	days := []HistoryDay{{Date: "2025-11-21", Tokens: 172043, Cost: 0.31}}

	if err := SaveHistory(path, days); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := SaveHistory(path, nil); err != nil {
		t.Fatalf("save empty: %v", err)
	}

	if got := LoadHistory(path); len(got) != 1 {
		t.Errorf("history after an empty save = %v, want the stored day kept", got)
	}
}

func TestHistoryRoundTripsThroughDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "ai-history.json")
	days := []HistoryDay{
		{Date: "2025-11-21", Tokens: 172043, Cost: 0.31, Agents: []HistoryAgentDay{{Agent: "gemini", Tokens: 172043, Cost: 0.31}}},
		{Date: "2026-08-13", Tokens: 54149914, Cost: 43.32, Agents: []HistoryAgentDay{{Agent: "claude", Tokens: 54149914, Cost: 43.32}}},
	}

	if err := SaveHistory(path, days); err != nil {
		t.Fatalf("save: %v", err)
	}

	got := LoadHistory(path)
	if len(got) != 2 {
		t.Fatalf("loaded %d days, want 2: %v", len(got), got)
	}
	if got[1].Cost != 43.32 {
		t.Errorf("cost = %v, want 43.32", got[1].Cost)
	}
}

// A missing file is the normal first-run state, not an error.
func TestLoadingAbsentFilesYieldsNothing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.json")

	if days := LoadHistory(missing); days != nil {
		t.Errorf("history = %v, want nil", days)
	}
	if _, ok := LoadQuotaSnapshot(missing); ok {
		t.Error("quota snapshot reported ok for a missing file")
	}
	if days := LoadHistory(""); days != nil {
		t.Errorf("empty path yielded %v", days)
	}
}

// The temp file is created in the destination directory, so a failure to clean
// up leaves litter next to the real file -- and the next Load would not see it,
// making the leak invisible until the directory is inspected by hand.
func TestAtomicWriteLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai-history.json")

	for range 3 {
		if err := SaveHistory(path, []HistoryDay{{Date: "2026-08-13", Tokens: 1}}); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "ai-history.json" {
			t.Errorf("left behind %q", entry.Name())
		}
	}
}

// Overwriting must replace the file wholesale. A shorter new payload written
// over a longer old one without truncation would leave trailing bytes and
// produce JSON that parses as the OLD content plus garbage.
func TestAtomicWriteFullyReplacesALongerFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-history.json")

	long := make([]HistoryDay, 0, 40)
	for day := 1; day <= 28; day++ {
		long = append(long, HistoryDay{
			Date:   "2026-02-" + string(rune('0'+day/10)) + string(rune('0'+day%10)),
			Tokens: 1_000_000,
		})
	}
	if err := SaveHistory(path, long); err != nil {
		t.Fatalf("save long: %v", err)
	}
	if err := SaveHistory(path, []HistoryDay{{Date: "2026-08-13", Tokens: 7}}); err != nil {
		t.Fatalf("save short: %v", err)
	}

	got := LoadHistory(path)
	if len(got) != 1 || got[0].Date != "2026-08-13" {
		t.Errorf("loaded %v, want only the short write", got)
	}
}
