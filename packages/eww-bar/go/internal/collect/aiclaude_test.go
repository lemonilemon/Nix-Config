package collect

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const (
	claudeTestNow      = 1_800_000_000.0
	claudeTestSnapshot = "/state/usage-snapshot.json"
	claudeTestCreds    = "/cfg/.credentials.json"
)

func claudeTestISO(epoch float64) string {
	return time.Unix(int64(epoch), 0).UTC().Format("2006-01-02T15:04:05.000Z")
}

func claudeTestCredentials(expiresAt float64) string {
	return fmt.Sprintf(`{"claudeAiOauth":{"accessToken":"token","expiresAt":%d,"subscriptionType":"max"}}`,
		int64(expiresAt*1000))
}

func claudeTestSnapshotJSON(at, fivePercent, fiveReset, sevenPercent, sevenReset float64) string {
	return fmt.Sprintf(`{"updated_at":%q,"five_hour":{"used_percentage":%g,"resets_at":%q},"seven_day":{"used_percentage":%g,"resets_at":%q}}`,
		claudeTestISO(at), fivePercent, claudeTestISO(fiveReset), sevenPercent, claudeTestISO(sevenReset))
}

var claudeTestAPIBody = fmt.Sprintf(`{"five_hour":{"utilization":50,"resets_at":%q},"seven_day":{"utilization":60,"resets_at":%q}}`,
	claudeTestISO(claudeTestNow+3*3600), claudeTestISO(claudeTestNow+3*86400))

// installClaudeFixture returns a counter of usage API requests.
func installClaudeFixture(t *testing.T, fixture map[string]string) *int {
	t.Helper()
	fixture["now"] = fmt.Sprintf("%.0f", claudeTestNow)
	fixture["env.CLAUDE_CONFIG_DIR"] = "/cfg"
	restore := InstallFixture(fixture)
	ResetQuotaCache()

	requests := 0
	fixtureHTTP := httpGetText
	httpGetText = func(url string, headers map[string]string, timeout time.Duration) (int, string, error) {
		requests++
		return fixtureHTTP(url, headers, timeout)
	}
	t.Cleanup(func() {
		httpGetText = fixtureHTTP
		restore()
		ResetQuotaCache()
	})
	return &requests
}

func windowPercents(card Quota) []int {
	percents := make([]int, len(card.Windows))
	for i, window := range card.Windows {
		percents[i] = window.Percent
	}
	return percents
}

func TestClaudeCardUsesAFreshSnapshotWithoutTheAPI(t *testing.T) {
	takenAt := claudeTestNow - 120
	requests := installClaudeFixture(t, map[string]string{
		"file\x1f" + claudeTestCreds:    claudeTestCredentials(claudeTestNow + 3600),
		"file\x1f" + claudeTestSnapshot: claudeTestSnapshotJSON(takenAt, 23, claudeTestNow+3600, 41, claudeTestNow+86400),
		"http.body":                     claudeTestAPIBody,
	})

	card := ClaudeCard(claudeTestSnapshot, true, ClaudeQuotaDefault("waiting"))
	if *requests != 0 {
		t.Fatalf("usage API requests = %d, want 0 for a fresh snapshot", *requests)
	}
	if card.Status != "live" || card.Updated != clockHHMM(takenAt) || card.Plan != "Max" {
		t.Fatalf("card = %+v, want live, updated %s, plan Max", card, clockHHMM(takenAt))
	}
	if got := windowPercents(card); fmt.Sprint(got) != "[23 41]" {
		t.Fatalf("window percents = %v, want [23 41]", got)
	}
}

func TestClaudeCardAsksTheAPIWhenTheSnapshotIsOld(t *testing.T) {
	requests := installClaudeFixture(t, map[string]string{
		"file\x1f" + claudeTestCreds:    claudeTestCredentials(claudeTestNow + 3600),
		"file\x1f" + claudeTestSnapshot: claudeTestSnapshotJSON(claudeTestNow-20*60, 23, claudeTestNow+3600, 41, claudeTestNow+86400),
		"http.body":                     claudeTestAPIBody,
	})

	card := ClaudeCard(claudeTestSnapshot, true, ClaudeQuotaDefault("waiting"))
	if *requests != 1 {
		t.Fatalf("usage API requests = %d, want 1", *requests)
	}
	if got := windowPercents(card); fmt.Sprint(got) != "[50 60]" || card.Updated != clockHHMM(claudeTestNow) {
		t.Fatalf("card = %+v, want the API's [50 60] updated now", card)
	}
}

func TestClaudeCardLeavesTheAPIAloneOffTheProbeTick(t *testing.T) {
	requests := installClaudeFixture(t, map[string]string{
		"file\x1f" + claudeTestCreds:    claudeTestCredentials(claudeTestNow + 3600),
		"file\x1f" + claudeTestSnapshot: claudeTestSnapshotJSON(claudeTestNow-20*60, 23, claudeTestNow+3600, 41, claudeTestNow+86400),
		"http.body":                     claudeTestAPIBody,
	})

	card := ClaudeCard(claudeTestSnapshot, false, ClaudeQuotaDefault("waiting"))
	if *requests != 0 {
		t.Fatalf("usage API requests = %d, want 0 without allowAPI", *requests)
	}
	if card.Status != "live" || fmt.Sprint(windowPercents(card)) != "[23 41]" {
		t.Fatalf("card = %+v, want the snapshot, still live", card)
	}
}

func TestClaudeCardPrefersTheNewerAPIReading(t *testing.T) {
	requests := installClaudeFixture(t, map[string]string{
		"file\x1f" + claudeTestCreds:    claudeTestCredentials(claudeTestNow + 3600),
		"file\x1f" + claudeTestSnapshot: claudeTestSnapshotJSON(claudeTestNow-20*60, 23, claudeTestNow+3600, 41, claudeTestNow+86400),
		"http.body":                     claudeTestAPIBody,
	})

	ClaudeCard(claudeTestSnapshot, true, ClaudeQuotaDefault("waiting"))
	card := ClaudeCard(claudeTestSnapshot, false, ClaudeQuotaDefault("waiting"))
	if *requests != 1 {
		t.Fatalf("usage API requests = %d, want 1 across both calls", *requests)
	}
	if got := windowPercents(card); fmt.Sprint(got) != "[50 60]" {
		t.Fatalf("window percents = %v, want the newer API reading [50 60]", got)
	}
}

func TestClaudeCardShowsAnOldReadingAsOfWithPassedResetsAtZero(t *testing.T) {
	takenAt := claudeTestNow - 2*3600
	requests := installClaudeFixture(t, map[string]string{
		"file\x1f" + claudeTestCreds:    claudeTestCredentials(claudeTestNow - 60),
		"file\x1f" + claudeTestSnapshot: claudeTestSnapshotJSON(takenAt, 40, claudeTestNow-3600, 41, claudeTestNow+3*86400),
	})

	card := ClaudeCard(claudeTestSnapshot, true, ClaudeQuotaDefault("waiting"))
	if *requests != 0 {
		t.Fatalf("usage API requests = %d, want 0 with an expired token", *requests)
	}
	if want := "as of " + restoredClock(takenAt, claudeTestNow); card.Status != want {
		t.Fatalf("status = %q, want %q", card.Status, want)
	}
	session, weekly := card.Windows[0], card.Windows[1]
	if session.Percent != 0 || session.Remaining != "--" || session.Reset != "--" {
		t.Fatalf("session window = %+v, want 0%% with no countdown after its reset", session)
	}
	if weekly.Percent != 41 || weekly.Remaining != FormatRemaining(3*86400) {
		t.Fatalf("weekly window = %+v, want 41%% resetting in %s", weekly, FormatRemaining(3*86400))
	}
}

func TestClaudeCardWithoutAReadingKeepsTheAPIFailure(t *testing.T) {
	installClaudeFixture(t, map[string]string{
		"file\x1f" + claudeTestCreds: claudeTestCredentials(claudeTestNow - 60),
	})

	if card := ClaudeCard(claudeTestSnapshot, true, ClaudeQuotaDefault("waiting")); card.Status != statusTokenExpired {
		t.Fatalf("status = %q, want %q", card.Status, statusTokenExpired)
	}
	previous := ClaudeQuotaDefault("as of Jan 2")
	if card := ClaudeCard(claudeTestSnapshot, false, previous); card.Status != previous.Status {
		t.Fatalf("status = %q, want the previous card's %q", card.Status, previous.Status)
	}
}

func TestClaudeSnapshotReadingRejectsUnusableFiles(t *testing.T) {
	for name, text := range map[string]string{
		"garbage":       "{not json",
		"no timestamp":  `{"five_hour":{"used_percentage":10,"resets_at":null}}`,
		"no windows":    fmt.Sprintf(`{"updated_at":%q}`, claudeTestISO(claudeTestNow)),
		"null percents": fmt.Sprintf(`{"updated_at":%q,"five_hour":{"used_percentage":null,"resets_at":null}}`, claudeTestISO(claudeTestNow)),
	} {
		t.Run(name, func(t *testing.T) {
			installClaudeFixture(t, map[string]string{"file\x1f" + claudeTestSnapshot: text})
			if reading, ok := ClaudeSnapshotReading(claudeTestSnapshot, claudeTestNow); ok {
				t.Fatalf("got a reading %+v, want none", reading)
			}
		})
	}
}

func TestQuotaStatesWithSnapshotProbesOpenusageOnlyOnRefresh(t *testing.T) {
	installClaudeFixture(t, map[string]string{
		"env.EWW_BAR_CLAUDE_USAGE_SNAPSHOT": claudeTestSnapshot,
		"file\x1f" + claudeTestCreds:        claudeTestCredentials(claudeTestNow + 3600),
		"file\x1f" + claudeTestSnapshot:     claudeTestSnapshotJSON(claudeTestNow-60, 23, claudeTestNow+3600, 41, claudeTestNow+86400),
	})
	probes := func() int {
		count := 0
		for _, entry := range FixtureJournal() {
			if strings.HasPrefix(entry, "run\x1fopenusage-cli\x1fprobe") {
				count++
			}
		}
		return count
	}

	first := QuotaStates(true)
	second := QuotaStates(false)
	if probes() != 1 {
		t.Fatalf("openusage probes = %d, want 1: only the refresh probes", probes())
	}
	if len(first) != 1+len(openusageProviders) || len(second) != len(first) {
		t.Fatalf("card counts = %d then %d, want %d", len(first), len(second), 1+len(openusageProviders))
	}
	if second[0].Key != "claude" || fmt.Sprint(windowPercents(second[0])) != "[23 41]" {
		t.Fatalf("claude card = %+v, want it rebuilt from the snapshot", second[0])
	}
}
