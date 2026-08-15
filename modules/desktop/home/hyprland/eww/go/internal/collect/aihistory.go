package collect

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// The durable half of the AI subsystem: a year of daily usage, kept on disk so
// the popup has something to draw the moment the bar starts and so history
// outlives the transcripts it was derived from.
//
// Why a file and not a database. Every import in this module is stdlib, which
// is what lets default.nix set vendorHash = null and carry no fetch derivation
// at all. A SQLite driver is either cgo or five megabytes of translated C, and
// either one ends that for a table this size: a year of daily rows is ~365
// records of five fields. sqlite3 is not in runtimePackages either, so shelling
// out is not available without adding it.
//
// Why it exists at all, given ccusage reads durable logs. Measured on this host
// ccusage reaches back to 2025-11-21, which is well past Claude Code's
// cleanupPeriodDays default of 30 -- but that default is the agents' to change,
// not ours, and a pruned transcript is unrecoverable. Snapshotting the daily
// rollup costs a few kilobytes and makes the horizon ours.

// HistoryAgentDay is one agent's share of one day.
//
// Real per-agent numbers, not a list of names. ccusage's --by-agent report
// carries a full breakdown under each daily row's "agents" array, and the
// difference is not cosmetic: crediting a shared day's whole total to every
// agent that touched it made the five agent rows sum to 4517M under a
// day-summed total of 2943M, so the summary visibly failed to add up.
type HistoryAgentDay struct {
	Agent  string  `json:"agent"`
	Tokens float64 `json:"tokens"`
	Cost   float64 `json:"cost"`
}

// HistoryDay is one day's usage across every agent, as stored.
//
// Tokens and Cost are the raw numbers rather than formatted strings: this is a
// storage record that gets re-bucketed against changing neighbours every time
// the grid is rebuilt, so it has to stay arithmetic.
type HistoryDay struct {
	Date   string            `json:"date"` // YYYY-MM-DD, local zone, as ccusage reports it
	Tokens float64           `json:"tokens"`
	Cost   float64           `json:"cost"`
	Agents []HistoryAgentDay `json:"agents"`
}

// HistoryStore is the on-disk file.
//
// Version is not ceremony. The grid derives from these records, so a later
// change to what a record means has to be able to tell old rows from new ones
// without guessing, and the alternative to a number is sniffing field presence.
type HistoryStore struct {
	Version int          `json:"version"`
	Days    []HistoryDay `json:"days"`
}

// historyVersion is the current HistoryStore.Version.
const historyVersion = 1

// HistoryDaysFromJSON reads the daily section of a ccusage report.
//
// Takes the report text rather than reading it, matching BatteryStateFromFiles
// and ClaudeQuotaStateFromJSON: the parsing is what has edge cases worth
// testing, and the subprocess is not.
//
// A report that does not parse yields no days rather than an error. Every
// caller's response to a broken report is the same -- keep what is already
// stored -- and an empty slice expresses that without a second return value.
func HistoryDaysFromJSON(report string) []HistoryDay {
	var body struct {
		Daily []map[string]any `json:"daily"`
	}
	decoder := json.NewDecoder(strings.NewReader(report))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return nil
	}

	days := make([]HistoryDay, 0, len(body.Daily))
	for _, row := range body.Daily {
		date, isText := row["period"].(string)
		if !isText || !isCalendarDate(date) {
			// The same report shape carries weekly and monthly rollups, whose
			// period is "2026-08-03" (week start) and "2026-08". Only the
			// daily section is read here, but a caller pointing this at the
			// wrong section would otherwise silently mix rollups into the grid
			// and inflate every bucket it touched.
			continue
		}

		values := DailyTokenValues(row)
		days = append(days, HistoryDay{
			Date:   date,
			Tokens: values.Total,
			Cost:   values.Cost,
			Agents: dayAgents(row),
		})
	}

	sortHistory(days)
	return days
}

// UnpricedModels names every model that burned tokens and cost nothing.
//
// The backstop for the pricing overrides in overlays/default.nix. ccusage
// prices a model its table does not know at exactly zero, and in --json mode it
// says nothing at all about having done so -- the missing-pricing warning only
// appears in table mode, which nothing here reads. That is how claude-opus-5
// reported $616 of spend as free for weeks with no signal anywhere.
//
// The override list is hand-maintained and the next model launch will age it
// out again, so the condition is checked rather than the model list: tokens
// spent with a cost of zero is not a state any priced model can reach.
//
// Names come from modelBreakdowns rather than the day total because the name is
// what makes the warning actionable -- it is the key to add to the override.
func UnpricedModels(report string) []string {
	var body struct {
		Daily []map[string]any `json:"daily"`
	}
	decoder := json.NewDecoder(strings.NewReader(report))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return nil
	}

	seen := map[string]bool{}
	for _, row := range body.Daily {
		for _, raw := range ListValue(row, "modelBreakdowns") {
			breakdown, isMap := raw.(map[string]any)
			if !isMap {
				continue
			}
			name, isText := breakdown["modelName"].(string)
			if !isText || name == "" {
				continue
			}
			if NumberValue(breakdown, "cost") > 0 {
				continue
			}
			if DailyTokenValues(breakdown).Total > 0 {
				seen[name] = true
			}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// isCalendarDate reports whether text is exactly YYYY-MM-DD and names a real
// day. time.Parse alone would accept "2026-8-3", which sorts wrong as a string
// and would land in the grid under the wrong column.
func isCalendarDate(text string) bool {
	if len(text) != len("2006-01-02") {
		return false
	}
	parsed, err := time.Parse("2006-01-02", text)
	return err == nil && parsed.Format("2006-01-02") == text
}

// dayAgents is the per-agent breakdown for one daily row, sorted by name.
//
// Sorted rather than left in report order because ccusage does not promise a
// stable order across runs and these records are compared and diffed on disk.
//
// The "all" pseudo-agent is skipped: it is the row's own total wearing an agent
// label, and summing it beside the real ones would double every figure.
func dayAgents(row map[string]any) []HistoryAgentDay {
	entries := ListValue(row, "agents")
	agents := make([]HistoryAgentDay, 0, len(entries))
	for _, entry := range entries {
		fields, isMap := entry.(map[string]any)
		if !isMap {
			continue
		}
		name, isText := fields["agent"].(string)
		if !isText || name == "" || name == "all" {
			continue
		}
		values := DailyTokenValues(fields)
		agents = append(agents, HistoryAgentDay{
			Agent:  PyLower(name),
			Tokens: values.Total,
			Cost:   values.Cost,
		})
	}
	sort.Slice(agents, func(a, b int) bool { return agents[a].Agent < agents[b].Agent })
	return agents
}

// sortHistory orders days oldest first. String order is date order for
// YYYY-MM-DD, which is why isCalendarDate insists on the zero padding.
func sortHistory(days []HistoryDay) {
	sort.Slice(days, func(a, b int) bool { return days[a].Date < days[b].Date })
}

// MergeHistoryDays folds a fresh ccusage report into what is already stored.
//
// Fresh wins on every field but one, because it is the newer read of the same
// underlying logs and today's row keeps growing all day. Dates only stored have
// been pruned from the logs and are exactly what the file exists to preserve,
// so they are kept untouched.
//
// The exception is cost. A day can lose its price and keep its tokens: ccusage
// prices a model it does not know at zero, so a pricing table that goes
// backwards -- an --offline fallback on a flaky network, a model renamed
// upstream -- would otherwise overwrite real money with $0.00 permanently. When
// the token count agrees and only the cost has collapsed to zero, the stored
// price is the better number and is kept.
func MergeHistoryDays(stored, fresh []HistoryDay) []HistoryDay {
	byDate := make(map[string]HistoryDay, len(stored)+len(fresh))
	for _, day := range stored {
		byDate[day.Date] = day
	}

	for _, day := range fresh {
		if previous, existed := byDate[day.Date]; existed {
			if day.Cost == 0 && previous.Cost > 0 && day.Tokens == previous.Tokens {
				day.Cost = previous.Cost
			}
		}
		byDate[day.Date] = day
	}

	merged := make([]HistoryDay, 0, len(byDate))
	for _, day := range byDate {
		merged = append(merged, day)
	}
	sortHistory(merged)
	return merged
}

// ParseHistoryStore decodes the on-disk file.
//
// Unreadable, truncated or future-versioned content yields no days. That is the
// safe direction: the caller merges fresh ccusage output into whatever comes
// back, so a corrupt file costs the pruned tail and nothing else, while
// treating a parse failure as fatal would blank a working bar.
func ParseHistoryStore(text string) []HistoryDay {
	var store HistoryStore
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := decoder.Decode(&store); err != nil {
		return nil
	}
	if store.Version > historyVersion {
		return nil
	}

	days := make([]HistoryDay, 0, len(store.Days))
	for _, day := range store.Days {
		if isCalendarDate(day.Date) {
			days = append(days, day)
		}
	}
	sortHistory(days)
	return days
}

// EncodeHistoryStore renders the file's contents.
func EncodeHistoryStore(days []HistoryDay) (string, error) {
	if days == nil {
		days = []HistoryDay{}
	}
	// Indented, and that is a real choice rather than a default. This file is
	// the only durable state the bar keeps; when something looks wrong in the
	// grid the first move is to open it, and a single-line year of dates cannot
	// be read or usefully diffed.
	encoded, err := json.MarshalIndent(HistoryStore{
		Version: historyVersion,
		Days:    days,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded) + "\n", nil
}
