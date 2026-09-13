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

// openusageProviders mirrors OPENUSAGE_PROVIDERS.
//
// Claude is deliberately NOT in this list and stays on the read-only collector
// below. Its openusage plugin performs a full OAuth refresh, stores a ROTATED
// refresh token, and rewrites ~/.claude/.credentials.json non-atomically; `probe`
// has no read-only mode. A spent refresh token replayed by Claude Code is what
// OAuth 2.1 tells providers to treat as a breach. Re-evaluate only if openusage
// grows a read-only probe mode.
var openusageProviders = []struct{ Key, Name string }{
	{"codex", "Codex"},
	{"antigravity", "Antigravity"},
}

const (
	claudeUsageURL  = "https://api.anthropic.com/api/oauth/usage"
	claudeOAuthBeta = "oauth-2025-04-20"

	// The two failure statuses that carry a U+2014 em dash. Escaped because a
	// hyphen would look identical in review.
	statusTokenExpired = "token expired \u2014 open Claude Code"
	statusUnauthorized = "unauthorized \u2014 open Claude Code"

	// 120 s per provider. A timeout throws away the entire probe, and the probe
	// is off the five-minute poll anyway (see QuotaStates), so a generous
	// ceiling costs nothing.
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

// ClaudeLoadOAuth is READ ONLY, a hard constraint. Claude Code rotates this token;
// a second writer racing over the refresh token can invalidate the login, so this
// uses the access token as-is and reports a stale status instead of refreshing.
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
		// Everything but 401/403 maps to the generic status.
		return ClaudeQuotaDefault("unavailable")
	}

	plan, _ := oauth["subscriptionType"]
	if plan == nil {
		plan = "--"
	}
	return ClaudeQuotaStateFromJSON(body, plan, 0)
}

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

	// Keyed by providerId, and only string ids are indexed: every lookup below
	// uses a string key, so skipping non-strings gives the same answers.
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

// ResetQuotaCache drops the cached provider cards. For tests.
func ResetQuotaCache() {
	quotaCacheLock.Lock()
	defer quotaCacheLock.Unlock()
	cachedQuotas = nil
}

// QuotaStates is the per-provider cards the AI popup renders: cached, and
// deliberately kept off the background poll.
//
// None of what it produces reaches the bar face -- eww.yuck renders only
// ai_usage.text, from ccusage -- and the popup already sends `eww-barctl ai
// refresh` when it opens, so the probe runs when someone is looking. A cold cache
// always probes, so startup does not leave the popup blank until the backstop.
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

// ccusageHistoryDays is how far back the History tab's report reaches. Two years,
// well past the 371 days the grid can draw, because ccusage walks every transcript
// whatever the window, so --since only trims the output.
const ccusageHistoryDays = 730

// AiHistoryState folds the fresh report into the durable store and writes it back.
// Reads before it writes and merges rather than replaces, which makes the file a
// ratchet: a day that has aged out of the transcripts survives in the store.
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
	// Failure here is deliberately not propagated: the grid is already correct in
	// memory, and a read-only state directory should cost persistence, not the tab.
	_ = SaveHistory(path, merged)

	history := HeatmapFromDays(merged, 0)
	// From the fresh report, not the merged store: the question worth answering
	// is whether the CURRENT binary can price what is being run today.
	history.Warning = WarnUnpriced(UnpricedModels(report))
	return history
}

// SeedQuotaCache is separate from QuotaStates because that function's refresh=true
// path is pinned by recorded cases that expect a probe.
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

// CachedQuotas returns the cached cards without ever probing. The distinction from
// QuotaStates(false) is the point: that falls through to a probe on a cold cache,
// which is exactly what the fast startup path must not do.
func CachedQuotas() ([]Quota, bool) {
	quotaCacheLock.Lock()
	defer quotaCacheLock.Unlock()
	if cachedQuotas == nil {
		return nil, false
	}
	return append([]Quota(nil), cachedQuotas...), true
}

// AiUsageFast builds the bar's AI state from the ccusage report alone, because
// AiUsageState publishes once at the end and the quota half costs seconds while
// the ccusage half drives everything on the bar face.
//
// A new function rather than a change to AiUsageState or RefreshAiUsage: both are
// pinned by recorded cases, and RefreshAiUsage's record includes the exact SEQUENCE
// of values it publishes.
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

// RefreshAiUsage takes publish rather than the store directly because package state
// imports this one and the dependency cannot run both ways.
//
// refreshQuotas defaults to true at the callers that pass nothing -- the explicit
// ones behind `eww-barctl ai refresh`, which fire as the popup opens. Only the
// background poll opts out, via AiRefreshCycle.
func RefreshAiUsage(current AiUsage, publish func(AiUsage), refreshQuotas bool) (fresh AiUsage) {
	// Flag the refresh immediately so the popup shows a loading indicator over
	// the existing (still valid) data rather than blanking.
	pending := current
	if pending.Meta.Refreshing != "true" {
		pending.Meta.Refreshing = "true"
		publish(pending)
	}

	// Never leave the popup indicator pulsing after a transient failure. Every
	// failure path inside already returns a placeholder, so this recovers only
	// from a genuine bug -- but a stuck indicator is worse than a dropped refresh.
	defer func() {
		if recovered := recover(); recovered != nil {
			fresh = pending
			fresh.Meta.Refreshing = "false"
			publish(fresh)
		}
	}()

	fresh = AiUsageState(refreshQuotas)
	// Redundant today and kept deliberately: mutation testing correctly reports
	// removing this as inert, but it is the one place that guarantees the popup
	// indicator stops, and a permanently pulsing spinner is the worse failure.
	fresh.Meta.Refreshing = "false"
	publish(fresh)
	return fresh
}

// AiRefreshCycle is the callable the background AI poll runs, split by cost: the
// ccusage report drives the bar face and runs every cycle, while the openusage
// probe only feeds the popup and runs every quotaEvery-th, starting with the first
// so startup has quota cards. What the cadence buys is not holding the refresh
// cycle open for seconds, and not spending a provider's rate limit, into an empty
// room.
func AiRefreshCycle(quotaEvery int64) func(AiUsage, func(AiUsage)) AiUsage {
	// Atomic where the original uses itertools.count: a counter that silently
	// missed a tick would show up as the probe running at the wrong cadence.
	var counter atomic.Int64
	return func(current AiUsage, publish func(AiUsage)) AiUsage {
		tick := counter.Add(1) - 1
		return RefreshAiUsage(current, publish, tick%quotaEvery == 0)
	}
}
