package collect

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// The impure half of the AI subsystem: the two subprocesses, the one HTTP
// request, and the caches that keep them off the hot path.

// openusageProviders mirrors collectors.OPENUSAGE_PROVIDERS.
//
// Claude is deliberately NOT in this list and stays on the read-only collector
// below. Checked against the pinned openusage-community rev (eadcfe50), its
// plugins/claude/plugin.js performs a full OAuth refresh, stores a ROTATED
// refresh token, and writes ~/.claude/.credentials.json back with a bare
// non-atomic write; `openusage-cli probe` has no read-only mode, so there is no
// way to take its Claude number without granting that write access.
//
// Rotation is the hazard, not the write: a spent refresh token replayed by
// Claude Code is exactly the signal OAuth 2.1 tells providers to treat as a
// breach, and the documented response is revoking the whole token family.
// Re-evaluate only if openusage grows a read-only probe mode.
var openusageProviders = []struct{ Key, Name string }{
	{"codex", "Codex"},
	{"antigravity", "Antigravity"},
}

const (
	claudeUsageURL  = "https://api.anthropic.com/api/oauth/usage"
	claudeOAuthBeta = "oauth-2025-04-20"

	// The two failure statuses that carry a U+2014 em dash. Escaped because a
	// hyphen would look identical in review and would silently change what the
	// popup renders.
	statusTokenExpired = "token expired \u2014 open Claude Code"
	statusUnauthorized = "unauthorized \u2014 open Claude Code"

	// Measured on this host: the probe costs 41.6 s wall and 43.6 CPU-seconds.
	// The old 45 s ceiling left three seconds of headroom and a timeout here
	// throws away the entire probe. It is off the five-minute poll now (see
	// QuotaStates), so a generous ceiling costs nothing.
	openusageTimeout  = 120 * time.Second
	ccusageTimeout    = 20 * time.Second
	claudeHTTPTimeout = 10 * time.Second
)

// httpGetText is the seam for the backend's only HTTP request, matching RunText
// and ReadTextFile. Returns the status code and body; a transport failure comes
// back as a non-nil error with status 0.
var httpGetText = func(url string, headers map[string]string, timeout time.Duration) (int, string, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := (&http.Client{Timeout: timeout}).Do(request)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, "", err
	}
	return response.StatusCode, string(body), nil
}

// ClaudeCredentialsPaths mirrors collectors.claude_credentials_paths.
func ClaudeCredentialsPaths() []string {
	if configDir := Strip(os.Getenv("CLAUDE_CONFIG_DIR")); configDir != "" {
		return []string{filepath.Join(expandUser(configDir), ".credentials.json")}
	}
	return []string{
		expandUser("~/.claude/.credentials.json"),
		expandUser("~/.config/claude/.credentials.json"),
	}
}

// expandUser is pathlib's expanduser for the leading-~ case these paths use.
func expandUser(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

// ClaudeLoadOAuth mirrors collectors.claude_load_oauth.
//
// READ ONLY, and that is a hard constraint rather than an accident of the
// implementation. Claude Code rotates this token; a second writer racing over
// the refresh token can invalidate the login, so this reads the file, uses the
// access token as-is, and reports a stale status instead of ever refreshing.
// Nothing in this package may write to these paths.
func ClaudeLoadOAuth() (map[string]any, bool) {
	for _, path := range ClaudeCredentialsPaths() {
		text, ok := ReadTextFile(path)
		if !ok {
			continue
		}
		var data map[string]any
		if !ParseJSON(text, &data) {
			continue
		}
		oauth, isMap := data["claudeAiOauth"].(map[string]any)
		if isMap && pyTruthy(oauth["accessToken"]) {
			return oauth, true
		}
	}
	return nil, false
}

// ClaudeQuotaState mirrors collectors.claude_quota_state.
func ClaudeQuotaState() Quota {
	oauth, ok := ClaudeLoadOAuth()
	if !ok {
		return ClaudeQuotaDefault("not logged in")
	}

	// Some versions of the credentials file store milliseconds. The threshold
	// is the original's: anything past 1e12 cannot be a plausible second count.
	expiresAt := NumberValue(oauth, "expiresAt")
	if expiresAt > 1e12 {
		expiresAt /= 1000
	}
	if expiresAt > 0 && expiresAt < nowOr(0) {
		return ClaudeQuotaDefault(statusTokenExpired)
	}

	accessToken, _ := oauth["accessToken"].(string)
	status, body, err := httpGetText(claudeUsageURL, map[string]string{
		"Authorization":  "Bearer " + accessToken,
		"anthropic-beta": claudeOAuthBeta,
		"Accept":         "application/json",
		"User-Agent":     "eww-bar",
	}, claudeHTTPTimeout)

	switch {
	case err != nil:
		return ClaudeQuotaDefault("unavailable")
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ClaudeQuotaDefault(statusUnauthorized)
	case status >= 400:
		// urllib raises HTTPError for these and the original's handler maps
		// everything but 401/403 to the generic status.
		return ClaudeQuotaDefault("unavailable")
	}

	plan, _ := oauth["subscriptionType"]
	if plan == nil {
		plan = "--"
	}
	return ClaudeQuotaStateFromJSON(body, plan, 0)
}

// OpenusageQuotaStates mirrors collectors.openusage_quota_states.
func OpenusageQuotaStates(nowEpoch float64) []Quota {
	argv := []string{"probe"}
	for _, provider := range openusageProviders {
		argv = append(argv, provider.Key)
	}
	report := RunText(openusageTimeout, "openusage-cli", argv...)

	var snapshots []any
	if !ParseJSON(report, &snapshots) {
		snapshots = nil
	}

	// Keyed by providerId, and only string ids are indexed. The original builds
	// a dict keyed by whatever providerId holds, which raises TypeError on an
	// unhashable value; since every lookup below uses a string key, skipping
	// non-strings gives the same answers without the crash.
	byID := map[string]any{}
	for _, raw := range snapshots {
		snapshot, isMap := raw.(map[string]any)
		if !isMap {
			continue
		}
		if id, isText := snapshot["providerId"].(string); isText {
			byID[id] = snapshot
		}
	}

	quotas := make([]Quota, 0, len(openusageProviders))
	for _, provider := range openusageProviders {
		quotas = append(quotas,
			QuotaFromOpenusage(byID[provider.Key], provider.Key, provider.Name, nowEpoch))
	}
	return quotas
}

var (
	quotaCacheLock sync.Mutex
	cachedQuotas   []Quota
)

// ResetQuotaCache mirrors collectors.reset_quota_cache. For tests.
func ResetQuotaCache() {
	quotaCacheLock.Lock()
	defer quotaCacheLock.Unlock()
	cachedQuotas = nil
}

// QuotaStates mirrors collectors.quota_states: the per-provider cards the AI
// popup renders.
//
// Cached, and deliberately kept off the background poll. Measured on this host,
// `openusage-cli probe` costs 43.6 CPU-seconds and 41.6 s of wall clock per
// call against 0.85 for the ccusage report beside it -- the single largest
// consumer in the whole backend, by an order of magnitude.
//
// None of what it produces reaches the bar face: eww.yuck renders only
// ai_usage.text, which comes from ccusage. The quota cards live inside the
// popup, and the popup already sends `eww-barctl ai refresh` when it opens. So
// the probe runs when someone is actually looking, plus a slow backstop
// (AiRefreshCycle), instead of every five minutes into an empty room.
//
// A cold cache always probes, so startup does not leave the popup blank until
// the first backstop half an hour later.
func QuotaStates(refresh bool) []Quota {
	if !refresh {
		quotaCacheLock.Lock()
		if cachedQuotas != nil {
			cached := append([]Quota(nil), cachedQuotas...)
			quotaCacheLock.Unlock()
			return cached
		}
		quotaCacheLock.Unlock()
	}

	quotas := append([]Quota{ClaudeQuotaState()}, OpenusageQuotaStates(0)...)

	quotaCacheLock.Lock()
	cachedQuotas = append([]Quota(nil), quotas...)
	quotaCacheLock.Unlock()
	return quotas
}

var (
	lastAiUsageLock sync.Mutex
	lastAiUsage     *AiUsage
)

// ResetAiUsageCache drops the remembered last-good state. For tests.
func ResetAiUsageCache() {
	lastAiUsageLock.Lock()
	defer lastAiUsageLock.Unlock()
	lastAiUsage = nil
}

// ccusageSinceDays is how far back the report reaches. Long enough to cover the
// monthly rollup with room for a machine that has been off for a while.
const ccusageSinceDays = 45

// ccusageHistoryDays is how far back the History tab's report reaches.
//
// Two years, well past the 371 days the grid can draw, because the extra rows
// cost almost nothing and are kept: ccusage walks every transcript whatever the
// window, so --since only trims the output. Measured, 45 days costs 0.21 s and
// 365 costs 0.30 s. The wider ask exists so a first run captures everything the
// logs still hold before whatever prunes them gets there.
const ccusageHistoryDays = 730

// AiHistoryState collects the History tab's grid, folding the fresh report into
// the durable store and writing it back.
//
// Reads before it writes and merges rather than replaces, which is what makes
// the file a ratchet: a day that has aged out of the transcripts survives in
// the store, and a refresh that fails leaves what is already there alone.
func AiHistoryState(path string) AiHistory {
	since := localTime(nowOr(0) - ccusageHistoryDays*86400).Format("20060102")
	report := RunText(ccusageTimeout, "ccusage",
		"daily", "--json", "--offline",
		"--sections", "daily", "--by-agent", "--since", since,
	)

	merged := MergeHistoryDays(LoadHistory(path), HistoryDaysFromJSON(report))
	if len(merged) == 0 {
		return AiHistoryDefault()
	}
	// Failure here is deliberately not propagated. The grid the caller is about
	// to render is already correct in memory, and a read-only state directory
	// should cost persistence rather than the tab.
	_ = SaveHistory(path, merged)

	history := HeatmapFromDays(merged, 0)
	// From the fresh report, not the merged store: an old day priced at zero by
	// a table that has since learned the model is history, while the question
	// worth answering is whether the CURRENT binary can price what is being run
	// today.
	history.Warning = WarnUnpriced(UnpricedModels(report))
	return history
}

// SeedQuotaCache primes the quota cache from a snapshot so a cold start has
// cards to show.
//
// Separate from QuotaStates rather than folded into it, because QuotaStates'
// refresh=true path is pinned by recorded cases that expect a probe. This is
// only ever called by the daemon at startup, where no such expectation exists.
func SeedQuotaCache(quotas []Quota) {
	if len(quotas) == 0 {
		return
	}
	quotaCacheLock.Lock()
	defer quotaCacheLock.Unlock()
	if cachedQuotas == nil {
		cachedQuotas = append([]Quota(nil), quotas...)
	}
}

// CachedQuotas returns the cached cards without ever probing.
//
// The distinction from QuotaStates(false) is the whole point: that falls
// through to a 41.6 s probe on a cold cache, which is exactly what the fast
// startup path must not do.
func CachedQuotas() ([]Quota, bool) {
	quotaCacheLock.Lock()
	defer quotaCacheLock.Unlock()
	if cachedQuotas == nil {
		return nil, false
	}
	return append([]Quota(nil), cachedQuotas...), true
}

// AiUsageFast builds the bar's AI state from the ccusage report alone.
//
// This exists because AiUsageState computes ApplyQuotas(ccusage, QuotaStates())
// and publishes once, at the end. The ccusage half costs 0.21 s and drives
// everything on the bar face; the quota half drives only the popup's cards and
// costs anywhere from 6 s to the 41.6 s recorded at QuotaStates, depending on
// how the providers are feeling. So a cold start showed "-- " for all of that
// while the numbers for it sat finished in a local variable.
//
// A new function rather than a change to AiUsageState or RefreshAiUsage: both
// are pinned by recorded cases, and RefreshAiUsage's record includes the exact
// SEQUENCE of values it publishes, so adding an interim publish there would
// break it. The daemon calls this once at startup and the normal cycle takes
// over afterwards.
func AiUsageFast() AiUsage {
	since := localTime(nowOr(0) - ccusageSinceDays*86400).Format("20060102")
	report := RunText(ccusageTimeout, "ccusage",
		"daily", "--json", "--offline",
		"--sections", "daily,weekly,monthly",
		"--by-agent", "--since", since,
	)

	quotas, cached := CachedQuotas()
	if !cached {
		quotas = QuotaDefaults()
	}
	return ApplyQuotas(AiUsageStateFromJSON(report, 0), quotas)
}

// AiUsageState mirrors collectors.ai_usage_state.
func AiUsageState(refreshQuotas bool) AiUsage {
	since := localTime(nowOr(0) - ccusageSinceDays*86400).Format("20060102")
	report := RunText(ccusageTimeout, "ccusage",
		"daily", "--json", "--offline",
		"--sections", "daily,weekly,monthly",
		"--by-agent", "--since", since,
	)

	value := ApplyQuotas(AiUsageStateFromJSON(report, 0), QuotaStates(refreshQuotas))

	lastAiUsageLock.Lock()
	defer lastAiUsageLock.Unlock()

	if value.Source == "missing" && lastAiUsage != nil {
		// Keep showing the last good numbers rather than blanking the bar, but
		// mark them so the popup can say why they are not moving.
		stale := *lastAiUsage
		stale.Class = "stale"
		stale.Meta.Stale = "true"
		stale.Meta.Status = "stale"
		stale.Tooltip = "AI usage collector is stale\n" + stale.Tooltip
		return stale
	}
	if value.Source != "missing" {
		remembered := value
		lastAiUsage = &remembered
	}
	return value
}

// RefreshAiUsage mirrors collectors.refresh_ai_usage.
//
// publish is how the caller writes into BarState. Passed in rather than taking
// the store directly because package state imports this one, and the dependency
// cannot run both ways.
//
// refreshQuotas defaults to true at the callers that pass nothing -- the
// explicit ones, behind eww.yuck's `eww-barctl ai refresh`, which fire as the
// popup opens and are exactly when the expensive probe is worth paying for.
// Only the background poll opts out, via AiRefreshCycle.
func RefreshAiUsage(current AiUsage, publish func(AiUsage), refreshQuotas bool) (fresh AiUsage) {
	// Flag the refresh immediately so the popup shows a loading indicator over
	// the existing (still valid) data rather than blanking or sitting silent
	// while the combined ccusage + quota fetch runs.
	pending := current
	if pending.Meta.Refreshing != "true" {
		pending.Meta.Refreshing = "true"
		publish(pending)
	}

	// The original wraps the fetch in try/except for one reason, stated in its
	// comment: never leave the popup indicator pulsing after a transient
	// failure. Every failure path inside already returns a placeholder rather
	// than raising, so this recovers only from a genuine bug -- but the
	// guarantee it protects is a real one, and a stuck indicator is a worse
	// failure than a dropped refresh.
	defer func() {
		if recovered := recover(); recovered != nil {
			fresh = pending
			fresh.Meta.Refreshing = "false"
			publish(fresh)
		}
	}()

	fresh = AiUsageState(refreshQuotas)
	// Redundant today and kept deliberately: AiUsageState always builds from
	// AiUsageDefault or from a remembered state, both of which already carry
	// "false", so mutation testing correctly reports removing this as inert.
	// It stays because it is the one place that guarantees the popup indicator
	// stops, and a permanently pulsing spinner is a worse failure than the
	// assignment is a cost.
	fresh.Meta.Refreshing = "false"
	publish(fresh)
	return fresh
}

// AiRefreshCycle mirrors collectors.ai_refresh_cycle: the callable the
// background AI poll runs.
//
// Splits the refresh by cost. The ccusage report (0.85 CPU-seconds) drives the
// bar face and runs every cycle; the openusage probe (43.6) only feeds the
// popup and runs every quotaEvery-th, starting with the first so startup has
// quota cards. At the caller's 300 s period that is a probe every 30 minutes
// instead of every 5, on top of the on-demand refresh the popup already sends.
func AiRefreshCycle(quotaEvery int64) func(AiUsage, func(AiUsage)) AiUsage {
	// Atomic where the original uses itertools.count. Only one thread drives
	// this today, but a counter that silently misses a tick would show up as
	// the probe running at the wrong cadence, which is not something anyone
	// would notice from the bar.
	var counter atomic.Int64
	return func(current AiUsage, publish func(AiUsage)) AiUsage {
		tick := counter.Add(1) - 1
		return RefreshAiUsage(current, publish, tick%quotaEvery == 0)
	}
}
