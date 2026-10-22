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
// The zone is not incidental: restoredClock compares LOCAL calendar days, so the
// two constants above are same-day in UTC and straddle midnight at UTC+8. Package
// collect has no TestMain, so time.Local is CST+0800 on this host and UTC in the
// Nix sandbox -- pinning it is what keeps this from passing locally and failing in
// the build.
func pinClock(t *testing.T, epoch float64) {
	t.Helper()
	savedNow, savedZone := timeNow, time.Local
	timeNow = func() time.Time { return time.Unix(int64(epoch), 0) }
	time.Local = time.UTC
	t.Cleanup(func() { timeNow, time.Local = savedNow, savedZone })
}

// A card carries Updated as a bare HH:MM with no date, so restoring one unchanged
// would render yesterday's 14:32 as though the probe had just run.
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
	// eww.yuck renders quota.status verbatim once it is not "live".
	if claude.Status != "as of 14:32" {
		t.Errorf("status = %q, want \"as of 14:32\"", claude.Status)
	}
}

// A week-old snapshot rendered as a bare "as of 14:32" is the exact ambiguity
// this label exists to remove.
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

// A remembered failure is worse than no memory: "not logged in" restored at boot
// blames the user for a condition a since-completed login may have cleared.
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

// One good card and two failures would be restored next boot as a card set that
// never existed at the same moment.
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

// A refresh that found nothing must not be the thing that empties the file.
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

// The temp file is created in the destination directory, so a failure to clean up
// leaves litter the next Load would not see.
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

// A shorter new payload written over a longer old one without truncation would
// leave trailing bytes and parse as the OLD content plus garbage.
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
