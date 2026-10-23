package collect

import (
	"encoding/json"
	"os"
	"sync"
)

// The Claude card's second source: the rate limits claude-hud copies out of Claude
// Code's statusline input into display.externalUsageWritePath. Used only when
// EWW_BAR_CLAUDE_USAGE_SNAPSHOT names that file.

const (
	// claudeReadingFresh is seconds: a reading this young is shown without asking
	// the API.
	claudeReadingFresh = 5 * 60

	// claudeReadingLive is seconds: one quota probe period (AiRefreshCycle(6) at
	// 300 s in app.go) plus claudeReadingFresh. Older readings are shown "as of".
	claudeReadingLive = 6*300 + claudeReadingFresh
)

// claudeReading is one usage report in the usage API's shape, and when it was taken.
type claudeReading struct {
	Body string
	Plan any
	At   float64
}

var (
	claudeReadingLock    sync.Mutex
	lastClaudeAPIReading *claudeReading
)

func claudeUsageSnapshotPath() string {
	return Strip(os.Getenv("EWW_BAR_CLAUDE_USAGE_SNAPSHOT"))
}

// ClaudeSnapshotReading reads claude-hud's snapshot, renaming its used_percentage to
// the API's utilization so ClaudeQuotaStateFromJSON can build the card.
func ClaudeSnapshotReading(path string, nowEpoch float64) (claudeReading, bool) {
	text, ok := ReadTextFile(path)
	if !ok {
		return claudeReading{}, false
	}
	var snapshot map[string]any
	if !ParseJSON(text, &snapshot) {
		return claudeReading{}, false
	}
	at, parsed := ParseISOEpoch(snapshot["updated_at"])
	if !parsed || at <= 0 {
		return claudeReading{}, false
	}

	body := map[string]any{}
	for _, key := range []string{"five_hour", "seven_day"} {
		window, isMap := snapshot[key].(map[string]any)
		if !isMap || window["used_percentage"] == nil {
			continue
		}
		body[key] = map[string]any{
			"utilization": window["used_percentage"],
			"resets_at":   window["resets_at"],
		}
	}
	if len(body) == 0 {
		return claudeReading{}, false
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return claudeReading{}, false
	}

	var plan any = "--"
	if oauth, ok := ClaudeLoadOAuth(); ok {
		plan = claudePlan(oauth)
	}
	return claudeReading{Body: string(encoded), Plan: plan, At: min(at, nowOr(nowEpoch))}, true
}

// Card renders the reading at nowEpoch. A window whose reset has passed reads 0% with
// no countdown, and a reading older than claudeReadingLive says when it was taken.
func (reading claudeReading) Card(nowEpoch float64) Quota {
	now := nowOr(nowEpoch)
	card := ClaudeQuotaStateFromJSON(reading.Body, reading.Plan, now)
	if card.Status != "live" {
		return card
	}

	passed := claudeResetsPassed(reading.Body, now)
	for i, window := range card.Windows {
		if passed[window.Label] {
			card.Windows[i] = QuotaWindow{
				Label: window.Label, Percent: 0, Value: "0%",
				Remaining: "--", Reset: "--", Class: QuotaWindowClass(0, true),
			}
		}
	}
	card.Class = QuotaCardClass(card)
	card.Updated = clockHHMM(reading.At)
	if now-reading.At > claudeReadingLive {
		card.Status = "as of " + restoredClock(reading.At, now)
	}
	return card
}

// claudeResetsPassed names the windows, by card label, whose reset time is at or
// before now. It finds each window the way ClaudeQuotaStateFromJSON does.
func claudeResetsPassed(body string, now float64) map[string]bool {
	passed := map[string]bool{}
	var fields map[string]any
	if !ParseJSON(body, &fields) {
		return passed
	}
	for _, spec := range claudeWindowKeys {
		for _, key := range spec.Keys {
			window, isMap := fields[key].(map[string]any)
			if !isMap {
				continue
			}
			reset, parsed := ParseISOEpoch(pyOr(window["resets_at"], window["reset_at"]))
			if !parsed {
				reset = NumberValue(window, "resets_at", "reset_at")
			}
			if reset > 0 && reset <= now {
				passed[spec.Label] = true
			}
			break
		}
	}
	return passed
}

// ClaudeCard builds the Claude card from the newest reading: the snapshot at path, or
// the last API answer. With allowAPI it asks the API when neither is younger than
// claudeReadingFresh. With no reading at all it returns the API's failure card, or
// previous when the API was not asked.
func ClaudeCard(path string, allowAPI bool, previous Quota) Quota {
	now := nowOr(0)
	newest, have := ClaudeSnapshotReading(path, now)

	claudeReadingLock.Lock()
	if api := lastClaudeAPIReading; api != nil && (!have || api.At > newest.At) {
		newest, have = *api, true
	}
	claudeReadingLock.Unlock()

	if allowAPI && (!have || now-newest.At > claudeReadingFresh) {
		body, plan, failure, ok := claudeFetchUsage()
		if ok {
			fetched := claudeReading{Body: body, Plan: plan, At: now}
			card := fetched.Card(now)
			if card.Status == "live" {
				claudeReadingLock.Lock()
				lastClaudeAPIReading = &fetched
				claudeReadingLock.Unlock()
				return card
			}
			failure = card
		}
		if !have {
			return failure
		}
	}
	if !have {
		return previous
	}
	return newest.Card(now)
}

// quotaStatesWithSnapshot is QuotaStates when the snapshot is configured. The Claude
// card is rebuilt on every call because it costs a file read; the openusage cards
// keep QuotaStates' cache and its cold-cache probe.
func quotaStatesWithSnapshot(path string, refresh bool) []Quota {
	quotaCacheLock.Lock()
	cached := append([]Quota(nil), cachedQuotas...)
	probe := refresh || cachedQuotas == nil
	quotaCacheLock.Unlock()

	previous := ClaudeQuotaDefault("waiting")
	others := []Quota{}
	for _, card := range cached {
		if card.Key == "claude" {
			previous = card
		} else {
			others = append(others, card)
		}
	}
	if probe {
		others = OpenusageQuotaStates(0)
	}

	quotas := append([]Quota{ClaudeCard(path, probe, previous)}, others...)

	quotaCacheLock.Lock()
	cachedQuotas = append([]Quota(nil), quotas...)
	quotaCacheLock.Unlock()
	return quotas
}
