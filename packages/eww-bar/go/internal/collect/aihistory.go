package collect

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// The durable half of the AI subsystem: a year of daily usage, kept on disk so the
// popup has something to draw the moment the bar starts and so history outlives the
// transcripts it was derived from.
//
// A file and not a database because every import in this module is stdlib, which is
// what lets default.nix set vendorHash = null and carry no fetch derivation at all.
//
// It exists at all because a pruned transcript is unrecoverable: Claude Code's
// cleanupPeriodDays default is 30 and is the agents' to change, not ours.

// HistoryAgentDay is one agent's share of one day: real per-agent numbers from
// ccusage's --by-agent breakdown, not a list of names. Crediting a shared day's
// whole total to every agent made the rows sum to 4517M under a total of 2943M.
type HistoryAgentDay struct {
	Agent  string  `json:"agent"`
	Tokens float64 `json:"tokens"`
	Cost   float64 `json:"cost"`
}

// HistoryDay is one day's usage across every agent, as stored. Tokens and Cost stay
// raw numbers: this record is re-bucketed against changing neighbours every rebuild.
type HistoryDay struct {
	Date   string            `json:"date"` // YYYY-MM-DD, local zone, as ccusage reports it
	Tokens float64           `json:"tokens"`
	Cost   float64           `json:"cost"`
	Agents []HistoryAgentDay `json:"agents"`
}

// HistoryStore is the on-disk file. Version lets a later change to what a record
// means tell old rows from new ones without sniffing field presence.
type HistoryStore struct {
	Version int          `json:"version"`
	Days    []HistoryDay `json:"days"`
}

// historyVersion is the current HistoryStore.Version.
const historyVersion = 1

// HistoryDaysFromJSON reads the daily section of a ccusage report, taking the text
// rather than reading it. A report that does not parse yields no days rather than
// an error: every caller's response is the same, keep what is already stored.
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
			// The same report shape carries weekly and monthly rollups, whose period is
			// "2026-08-03" (week start) and "2026-08". A caller pointing this at the
			// wrong section would silently inflate every bucket it touched.
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

// UnpricedModels names every model that burned tokens and cost nothing: the
// backstop for the pricing overrides in overlays/default.nix.
//
// ccusage prices a model its table does not know at exactly zero, and says nothing
// about it in --json mode, which is how claude-opus-5 reported $616 of spend as
// free for weeks. The condition is checked rather than a model list, because tokens
// spent at zero cost is not a state any priced model can reach. Names come from
// modelBreakdowns because the name is the key to add to the override.
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

// isCalendarDate reports whether text is exactly YYYY-MM-DD and names a real day.
// time.Parse alone would accept "2026-8-3", which sorts wrong as a string.
func isCalendarDate(text string) bool {
	if len(text) != len("2006-01-02") {
		return false
	}
	parsed, err := time.Parse("2006-01-02", text)
	return err == nil && parsed.Format("2006-01-02") == text
}

// dayAgents is the per-agent breakdown for one daily row, sorted by name because
// ccusage does not promise a stable order and these records are diffed on disk. The
// "all" pseudo-agent is skipped: it is the row's own total wearing an agent label.
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

// MergeHistoryDays folds a fresh ccusage report into what is already stored. Fresh
// wins on every field but one; dates only stored have been pruned from the logs and
// are exactly what the file exists to preserve.
//
// The exception is cost. ccusage prices a model it does not know at zero, so a
// pricing table that goes backwards would otherwise overwrite real money with $0.00
// permanently. When the token count agrees and only the cost has collapsed to zero,
// the stored price is kept.
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

// ParseHistoryStore yields no days for unreadable, truncated or future-versioned
// content. The caller merges fresh ccusage output into whatever comes back, so a
// corrupt file costs the pruned tail, where a fatal parse would blank a working bar.
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
	// Indented deliberately: this is the only durable state the bar keeps, and a
	// single-line year of dates cannot be read or usefully diffed.
	encoded, err := json.MarshalIndent(HistoryStore{
		Version: historyVersion,
		Days:    days,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded) + "\n", nil
}
