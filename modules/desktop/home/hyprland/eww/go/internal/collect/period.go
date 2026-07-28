package collect

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// The periods layer: picking which ccusage row represents "today", "this week"
// and "this month", labelling its date range, and breaking it down per agent.

// Punctuation the AI popup renders. Escapes, not literals: an en dash is not a
// hyphen and a middle dot is not a period, and neither difference survives a
// careless edit or shows up in a diff.
const (
	enDash    = "\u2013" // week range separator
	middleDot = "\u00B7" // field separator in tooltips and agent lists
)

// weekSeconds is ccusage's week length. Seven fixed days, not a calendar week,
// which is why the week arithmetic below never touches a time.Time.
const weekSeconds = 7 * 86400

// agentDisplayNames mirrors collectors.AGENT_DISPLAY_NAMES.
var agentDisplayNames = map[string]string{
	"claude":   "Claude",
	"codex":    "Codex",
	"gemini":   "Gemini",
	"opencode": "OpenCode",
	"copilot":  "Copilot",
	"amp":      "Amp",
	"droid":    "Droid",
}

// AgentDisplayName mirrors collectors.agent_display_name.
func AgentDisplayName(key any) string {
	name, isText := key.(string)
	if !isText || name == "" {
		return "Unknown"
	}
	if display, known := agentDisplayNames[name]; known {
		return display
	}
	return PyTitle(strings.ReplaceAll(name, "_", " "))
}

// Date formats. Spelled out here because every one of them is a strftime
// directive in the original and the mapping is not guessable.
//
// The month and weekday NAMES are safe as fixed English. CPython does not call
// setlocale(LC_TIME, "") at startup and this backend never calls setlocale
// itself, so the process stays in the C locale and %B/%b/%a are English no
// matter what LC_TIME says -- verified by running the original under three
// other locales. Adding locale support later would break that equivalence,
// which is the whole reason this note is here.
const (
	layoutISODate   = "2006-01-02"   // %F
	layoutYearWeek  = "2006-01"      // %Y-%m
	layoutMonthYear = "January 2006" // %B %Y
)

// localTime is time.localtime(epoch): float seconds, floored.
func localTime(epoch float64) time.Time {
	return time.Unix(int64(epoch), 0).Local()
}

// CurrentPeriodKey mirrors collectors.current_period_key.
func CurrentPeriodKey(kind string, rows []map[string]any, now float64) string {
	if kind == "monthly" {
		return localTime(now).Format(layoutYearWeek)
	}
	if kind == "weekly" {
		// Align to ccusage's own week boundaries: step forward from the most
		// recent known week start in 7-day increments until `now` is covered,
		// rather than guessing which weekday a week starts on.
		latest, found := 0.0, false
		for _, row := range rows {
			start, ok := ParseISOEpoch(orDefault(row["period"], "") + "T00:00:00")
			if !ok {
				continue
			}
			if !found || start > latest {
				latest, found = start, true
			}
		}
		if found {
			for latest+weekSeconds <= now {
				latest += weekSeconds
			}
			return localTime(latest).Format(layoutISODate)
		}
	}
	return localTime(now).Format(layoutISODate)
}

// SyntheticPeriodRow mirrors collectors.synthetic_period_row: a zero-usage row
// keyed to the CURRENT period, so an idle day, week or month renders as itself
// with no usage rather than borrowing the last active row.
func SyntheticPeriodRow(kind string, rows []map[string]any, now float64) map[string]any {
	field := "date"
	if kind == "weekly" || kind == "monthly" {
		field = "period"
	}
	return map[string]any{field: CurrentPeriodKey(kind, rows, now)}
}

// SelectPeriodRow mirrors collectors.select_period_row.
func SelectPeriodRow(rows []any, kind string, nowEpoch float64) map[string]any {
	now := nowOr(nowEpoch)

	dictRows := []map[string]any{}
	for _, raw := range rows {
		if row, isMap := raw.(map[string]any); isMap {
			dictRows = append(dictRows, row)
		}
	}
	if len(dictRows) == 0 {
		return map[string]any{}
	}

	switch kind {
	case "monthly":
		current := localTime(now).Format(layoutYearWeek)
		for _, row := range dictRows {
			if strings.HasPrefix(orDefault(row["period"], ""), current) {
				return row
			}
		}
	case "weekly":
		for _, row := range dictRows {
			start, ok := ParseISOEpoch(orDefault(row["period"], "") + "T00:00:00")
			if ok && start <= now && now < start+weekSeconds {
				return row
			}
		}
	default:
		today := localTime(now).Format(layoutISODate)
		for _, row := range dictRows {
			if pyOr(row["period"], row["date"]) == today {
				return row
			}
		}
	}
	return SyntheticPeriodRow(kind, dictRows, now)
}

// PeriodRangeLabel mirrors collectors.period_range_label.
//
// ONE DIVERGENCE, deliberate. For kind "monthly" with a non-string period, the
// original's strptime raises TypeError and the except branch returns the value
// UNCHANGED -- so a numeric period leaks into the state as a JSON number where
// every other path yields a string. This stringifies instead, because AiPeriod
// types Range as a string and reproducing a latent type bug into a typed struct
// buys nothing. Unreachable from ccusage, which emits "2024-01"; pinned by
// test_period_range_label_leaks_a_non_string_month in the gate so the claim
// cannot rot.
func PeriodRangeLabel(kind string, period any, nowEpoch float64) string {
	if !pyTruthy(period) {
		return ""
	}

	if kind == "monthly" {
		text, isText := period.(string)
		if !isText {
			return pyStr(period)
		}
		if parsed, ok := parseYearMonth(text); ok {
			return parsed.Format(layoutMonthYear)
		}
		return text
	}

	start, ok := ParseISOEpoch(pyStr(period) + "T00:00:00")
	if !ok {
		return pyStr(period)
	}
	if kind == "weekly" {
		end := start + 6*86400
		return shortDate(start) + " " + enDash + " " + shortDate(end)
	}
	return localTime(start).Format("Mon ") + shortDate(start)
}

// shortDate is strftime("%b %-d"): the %-d is a glibc extension for an
// unpadded day, which Go has no layout for.
func shortDate(epoch float64) string {
	when := localTime(epoch)
	return when.Format("Jan ") + strconv.Itoa(when.Day())
}

// parseYearMonth is time.strptime(period, "%Y-%m"). Python's %m directive is a
// regex accepting one OR two digits, and the whole string must match.
func parseYearMonth(text string) (time.Time, bool) {
	rest := text
	year, rest, ok := takeDigits(rest, 4, 4)
	if !ok || !takeLiteral(&rest, '-') {
		return time.Time{}, false
	}
	month, rest, ok := takeDigits(rest, 1, 2)
	if !ok || month < 1 || month > 12 || rest != "" {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local), true
}

// PeriodAgents mirrors collectors.period_agents.
func PeriodAgents(row map[string]any) []PeriodAgent {
	total := NumberValue(row, "totalTokens", "total_tokens", "tokens")
	entries := ListValue(row, "agents")

	if total <= 0 {
		for _, raw := range entries {
			if entry, isMap := raw.(map[string]any); isMap {
				total += NumberValue(entry, "totalTokens", "total_tokens", "tokens")
			}
		}
	}

	agents := []PeriodAgent{}
	sortKeys := []float64{}
	for _, raw := range entries {
		entry, isMap := raw.(map[string]any)
		if !isMap {
			continue
		}
		key, isText := entry["agent"].(string)
		if !isText || key == "all" {
			continue
		}
		tokens := NumberValue(entry, "totalTokens", "total_tokens", "tokens")
		cost := NumberValue(entry, "totalCost", "total_cost", "cost")
		if tokens <= 0 && cost <= 0 {
			continue
		}
		lowered := PyLower(key)
		agents = append(agents, PeriodAgent{
			Key:     lowered,
			Name:    AgentDisplayName(lowered),
			Tokens:  FormatTokens(tokens),
			Cost:    FormatCost(cost),
			Percent: PercentPart(tokens, total),
		})
		sortKeys = append(sortKeys, tokens)
	}

	// list.sort is stable, and the key is the raw token count rather than the
	// formatted one -- "1.2M" and "1.2K" would compare as strings otherwise.
	order := make([]int, len(agents))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return sortKeys[order[a]] > sortKeys[order[b]]
	})
	sorted := make([]PeriodAgent, len(agents))
	for i, from := range order {
		sorted[i] = agents[from]
	}
	return sorted
}

// PeriodState mirrors collectors.period_state.
func PeriodState(rows []any, kind, label string, nowEpoch float64) AiPeriod {
	now := nowOr(nowEpoch)
	row := SelectPeriodRow(rows, kind, now)
	values := DailyTokenValues(row)

	// The original recomputes the input+output+cache fallback here when
	// values["total"] is non-positive. NOT ported: daily_token_values has
	// already applied exactly that fallback, so the second one can only ever
	// recompute the same number. Verified by exhausting the branch over 200k
	// generated rows; mutation testing is what surfaced it as unreachable.
	return AiPeriod{
		Label:  label,
		Range:  PeriodRangeLabel(kind, pyOr(pyOr(row["period"], row["date"]), ""), now),
		Tokens: FormatTokens(values.Total),
		Cost:   FormatCost(values.Cost),
		Agents: PeriodAgents(row),
	}
}

// ApplyQuotas mirrors collectors.apply_quotas.
//
// Takes and returns a value rather than mutating through a pointer: the
// original mutates its argument AND returns it, and every caller uses the
// return, so value semantics lose nothing and cannot alias.
func ApplyQuotas(state AiUsage, quotas []Quota) AiUsage {
	state.Quotas = quotas

	live := []Quota{}
	for _, quota := range quotas {
		if quota.Status == "live" {
			live = append(live, quota)
		}
	}

	if len(live) > 0 && state.Source == "missing" {
		state.Class = "active"
		state.Source = "quota"
		state.Meta.Status = "Quota only"
	}

	// Tint the bar amber when any subscription is near its limit.
	//
	// The original guards this with `source != "missing"`, which is dead: the
	// block above has already moved a missing source to "quota" whenever there
	// is a live quota to tint from, and with no live quota the loop does
	// nothing anyway. Dropped rather than reproduced, on the same evidence as
	// the fallback in PeriodState.
	for _, quota := range live {
		if quota.Class == "warning" || quota.Class == "critical" {
			state.Class = "warning"
			break
		}
	}

	// Rebuild the tooltip so hovering the bar leads with the glanceable thing:
	// each live subscription's headroom, then today's local spend.
	lines := []string{"AI usage"}
	for _, quota := range live {
		head := quota.Name
		if quota.Plan != "" && quota.Plan != "--" {
			head += " " + middleDot + " " + quota.Plan
		}
		for i, window := range quota.Windows {
			if i >= 2 {
				break
			}
			if window.Value != "--" {
				head += " " + middleDot + " " + window.Label + " " + window.Value
			}
		}
		lines = append(lines, head)
	}
	if state.Periods.Today.Tokens != "--" {
		lines = append(lines, "Today "+middleDot+" "+
			state.Periods.Today.Tokens+" / "+state.Periods.Today.Cost)
	}
	if len(lines) > 1 {
		state.Tooltip = strings.Join(lines, "\n")
	}
	return state
}

// AiUsageStateFromJSON mirrors collectors.ai_usage_state_from_json.
func AiUsageStateFromJSON(reportJSON string, nowEpoch float64) AiUsage {
	now := nowOr(nowEpoch)

	var report map[string]any
	if !ParseJSON(reportJSON, &report) {
		report = map[string]any{}
	}

	daily := ListValue(report, "daily")
	weekly := ListValue(report, "weekly")
	monthly := ListValue(report, "monthly")

	// A report is usable when ccusage returned any rows at all. A current day
	// with no recorded usage is valid data (Today = 0), not a missing
	// collector, so it must not fall through to the placeholder.
	if len(daily) == 0 && len(weekly) == 0 && len(monthly) == 0 {
		return AiUsageDefault()
	}

	state := AiUsageDefault()
	state.Updated = clockHHMM(now)
	state.Periods = AiPeriods{
		Today: PeriodState(daily, "daily", "Today", now),
		Week:  PeriodState(weekly, "weekly", "This week", now),
		Month: PeriodState(monthly, "monthly", "This month", now),
	}
	state.Text = glyphAiUsage + " " + state.Periods.Today.Tokens
	state.Agents = AgentsText(PeriodAgentKeys(SelectPeriodRow(daily, "daily", now)))
	state.Source = "ccusage"
	state.Meta.Status = "ccusage"
	state.Class = "active"
	state.Tooltip = "AI usage\nToday " + middleDot + " " +
		state.Periods.Today.Tokens + " / " + state.Periods.Today.Cost
	return state
}
