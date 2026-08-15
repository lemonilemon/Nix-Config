package collect

import (
	"fmt"
	"sort"
	"time"
)

// The contribution-graph grid behind the popup's History tab.
//
// Shape note, and it is load-bearing rather than stylistic: Columns is a plain
// [][]HeatCell rather than a slice of structs with a Cells field. eww's `for`
// body must be exactly one widget and a `for` may not be the direct body of
// another `for` (build_widget.rs rejects WidgetUse::Loop there), so the nesting
// has to be for -> box -> for, and the inner loop iterates the outer binding
// DIRECTLY. That is precisely how the wallpaper grid is built (eww.yuck:509-512
// over RowsFromItems), and matching it means the inner loop reads
// `(for cell in {column})` instead of `{column.cells}`.
//
// This whole structure reaches eww as its own variable via `eww update`, never
// through state.Bar. Two independent reasons, either sufficient: adding a field
// to AiUsage breaks 112 recorded golden cases across 7 entry points and one to
// state.Bar breaks 46, none of which can be regenerated; and eww rebuilds every
// child of a `for` whenever any variable the loop expression mentions changes,
// so a grid fed from bar_state would destroy and rebuild 424 widgets every time
// the CPU collector ticks, which is every 7 seconds.

// heatmapWeeks is how many week-columns the grid renders.
//
// 53 rather than 52 because 52 weeks is 364 days: a year plus the partial
// current week needs the extra column, which is the same reason GitHub's own
// graph is 53 wide.
const heatmapWeeks = 53

// heatLevels is how many non-zero shades the ramp has, matching the .level-N
// classes in eww.scss. Level 0 is the base .ai-heat-cell rule and emits no
// modifier class at all.
const heatLevels = 4

// HeatCell is one day.
//
// Class carries the whole visual state and Tooltip the whole hover text, so the
// yuck template stays a single interpolation of each with no logic. Anything
// conditional here would have to be a simplexpr ternary re-evaluated per cell
// per rebuild, 371 times.
type HeatCell struct {
	Class   string `json:"class"`
	Tooltip string `json:"tooltip"`
}

// HeatMonth is one month label above the grid.
//
// Width is a pixel count rather than a column span because the label row cannot
// use the grid's own layout: labels are variable-width text and the columns are
// fixed 5 px squares. Pinning each label to `span * pitch - gap` keeps the two
// rows aligned without a per-column label widget.
type HeatMonth struct {
	Label string `json:"label"`
	Width int    `json:"width"`
}

// HistoryAgent is one row of the per-agent summary under the grid.
type HistoryAgent struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Tokens string `json:"tokens"`
	Cost   string `json:"cost"`
}

// AiHistory is the ai_history eww variable.
//
// Warning is the only place the popup can admit its cost figures are
// incomplete. It is rendered here rather than on the bar's own AiUsage because
// that struct is pinned by 112 recorded cases, and because the History tab is
// where a wrong total is actually read.
type AiHistory struct {
	Columns [][]HeatCell   `json:"columns"`
	Months  []HeatMonth    `json:"months"`
	Agents  []HistoryAgent `json:"agents"`
	Tokens  string         `json:"tokens"`
	Cost    string         `json:"cost"`
	Range   string         `json:"range"`
	Days    int            `json:"days"`
	Status  string         `json:"status"`
	Warning string         `json:"warning"`
}

// heatCellPitch and heatCellGap mirror .ai-heat-cell's min-width and the grid
// boxes' :spacing in eww.scss. Duplicated here only to size the month labels;
// the cells themselves are sized entirely by CSS.
const (
	heatCellPitch = 10 // 8 px cell + 2 px spacing
	heatCellGap   = 2
)

// AiHistoryDefault is the value the ai_history defvar carries before the first
// collection, and what a failed one falls back to.
//
// An empty grid rather than a placeholder-filled one: the yuck renders whatever
// columns exist, so zero columns is an empty card and needs no "no data" branch
// in the template.
func AiHistoryDefault() AiHistory {
	return AiHistory{
		Columns: [][]HeatCell{},
		Months:  []HeatMonth{},
		Agents:  []HistoryAgent{},
		Tokens:  "--",
		Cost:    "--",
		Range:   "",
		Days:    0,
		Status:  "waiting",
		Warning: "",
	}
}

// WarnUnpriced renders the unpriced-model notice, empty when every model that
// spent tokens also has a price.
//
// Names the models rather than saying "some costs are missing", because the
// name is the key to add to the pricingOverrides table in overlays/default.nix
// and a warning that does not say what to do gets ignored.
func WarnUnpriced(models []string) string {
	switch len(models) {
	case 0:
		return ""
	case 1:
		return "No price for " + models[0]
	case 2:
		return "No price for " + models[0] + " and " + models[1]
	}
	return fmt.Sprintf("No price for %s and %d more", models[0], len(models)-1)
}

// heatThresholds returns the upper bound of each of the first heatLevels-1
// bands, computed as quantiles over the days that had any usage.
//
// Quantiles of non-zero days, not a fixed token scale, because the grid has to
// stay legible across wildly different usage. Measured on this host a heavy day
// is 300M tokens and a light one is 200K -- three orders of magnitude -- so any
// absolute ramp would either saturate every working day at level 4 or leave the
// light ones indistinguishable from empty. This is what GitHub does with commit
// counts and for the same reason.
//
// Zero days are excluded from the quantile: including them on a machine used
// twice a week would put the median at zero and push every active day to the
// top band.
func heatThresholds(days []HistoryDay) []float64 {
	active := make([]float64, 0, len(days))
	for _, day := range days {
		if day.Tokens > 0 {
			active = append(active, day.Tokens)
		}
	}
	if len(active) == 0 {
		return nil
	}
	sort.Float64s(active)

	thresholds := make([]float64, 0, heatLevels-1)
	for band := 1; band < heatLevels; band++ {
		// Nearest-rank on a 0-indexed slice. len-1 keeps the top band non-empty:
		// with index len the highest quantile would sit above every value and
		// level 4 could never be reached.
		index := band * (len(active) - 1) / heatLevels
		thresholds = append(thresholds, active[index])
	}
	return thresholds
}

// heatLevel buckets one day's tokens against the thresholds.
func heatLevel(tokens float64, thresholds []float64) int {
	if tokens <= 0 {
		return 0
	}
	level := 1
	for _, threshold := range thresholds {
		if tokens > threshold {
			level++
		}
	}
	if level > heatLevels {
		level = heatLevels
	}
	return level
}

// HeatmapFromDays builds the grid for the year ending on the day containing now.
//
// Columns run oldest to newest and each holds seven cells, Sunday first, which
// is the orientation GitHub uses and the one the month labels assume.
func HeatmapFromDays(days []HistoryDay, nowEpoch float64) AiHistory {
	history := AiHistoryDefault()
	if len(days) == 0 {
		return history
	}

	byDate := make(map[string]HistoryDay, len(days))
	for _, day := range days {
		byDate[day.Date] = day
	}
	thresholds := heatThresholds(days)

	today := localTime(nowOr(nowEpoch))
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// The grid ends on the Saturday of the current week, so today's column is
	// the last one and partially filled -- the same as GitHub's trailing edge.
	start := today.AddDate(0, 0, -int(today.Weekday())-(heatmapWeeks-1)*7)

	// Days before the first record are not "no usage", they are "not recorded",
	// and rendering them as an empty well would claim the machine sat idle
	// through months it was not being watched.
	sortHistory(days)
	firstRecorded := days[0].Date

	columns := make([][]HeatCell, 0, heatmapWeeks)
	monthSpans := make([]int, 0, 16)
	monthLabels := make([]string, 0, 16)

	for week := range heatmapWeeks {
		column := make([]HeatCell, 0, 7)
		for weekday := range 7 {
			date := start.AddDate(0, 0, week*7+weekday)
			column = append(column, heatCell(date, today, firstRecorded, byDate, thresholds))
		}
		columns = append(columns, column)

		// A month owns the columns whose Sunday falls inside it, which is the
		// convention that keeps each label at or before its month's first full
		// week rather than drifting a column late.
		label := start.AddDate(0, 0, week*7).Format("Jan")
		if len(monthLabels) == 0 || monthLabels[len(monthLabels)-1] != label {
			monthLabels = append(monthLabels, label)
			monthSpans = append(monthSpans, 1)
			continue
		}
		monthSpans[len(monthSpans)-1]++
	}

	history.Columns = columns
	history.Months = monthRow(monthLabels, monthSpans)
	history.Agents = historyAgents(days, start, today)
	history.Tokens, history.Cost, history.Days, history.Range = historyTotals(days, start, today)
	history.Status = "live"
	return history
}

// heatCell renders one day.
func heatCell(date, today time.Time, firstRecorded string,
	byDate map[string]HistoryDay, thresholds []float64) HeatCell {
	key := date.Format("2006-01-02")

	// Later than today, or earlier than anything recorded: no claim either way.
	if date.After(today) || key < firstRecorded {
		return HeatCell{Class: "blank"}
	}

	day, recorded := byDate[key]
	if !recorded || day.Tokens <= 0 {
		return HeatCell{
			Class:   "",
			Tooltip: date.Format("Mon 2 Jan 2006") + " · no usage",
		}
	}

	tooltip := fmt.Sprintf("%s · %s tokens", date.Format("Mon 2 Jan 2006"), FormatTokens(day.Tokens))
	if day.Cost > 0 {
		tooltip += " · " + FormatCost(day.Cost)
	}
	return HeatCell{
		Class:   fmt.Sprintf("level-%d", heatLevel(day.Tokens, thresholds)),
		Tooltip: tooltip,
	}
}

// monthRow turns column spans into pixel widths.
//
// A month is span*pitch wide, and only the LAST one loses a gap. The gap lives
// between columns, so a 53-column grid has 52 of them, not one per month --
// subtracting a gap from every month made the label row 12 px narrower than the
// grid, and since both are centred the difference split in half and offset
// every label by 6 px against the days it names.
//
// The first label is dropped when its month owns fewer than two columns: a
// one-column "Jan" at the left edge collides with the next label, and the
// leading partial month is the least informative one on the row.
func monthRow(labels []string, spans []int) []HeatMonth {
	months := make([]HeatMonth, 0, len(labels))
	for i, label := range labels {
		width := spans[i] * heatCellPitch
		if i == len(labels)-1 {
			width -= heatCellGap
		}
		if i == 0 && spans[i] < 2 {
			label = ""
		}
		months = append(months, HeatMonth{Label: label, Width: width})
	}
	return months
}

// historyAgents totals the window per agent, busiest first.
//
// Scoped to the same window as historyTotals, so the rows add up to the figure
// printed beside them. An earlier version summed the whole store here while the
// total covered one year, which made the two disagree on any machine with more
// than a year of history.
func historyAgents(days []HistoryDay, start, today time.Time) []HistoryAgent {
	from, to := start.Format("2006-01-02"), today.Format("2006-01-02")

	type total struct {
		tokens float64
		cost   float64
	}
	totals := map[string]*total{}
	for _, day := range days {
		if day.Date < from || day.Date > to {
			continue
		}
		for _, agent := range day.Agents {
			if totals[agent.Agent] == nil {
				totals[agent.Agent] = &total{}
			}
			totals[agent.Agent].tokens += agent.Tokens
			totals[agent.Agent].cost += agent.Cost
		}
	}

	agents := make([]HistoryAgent, 0, len(totals))
	for key, sum := range totals {
		agents = append(agents, HistoryAgent{
			Key:    key,
			Name:   AgentDisplayName(key),
			Tokens: FormatTokens(sum.tokens),
			Cost:   FormatCost(sum.cost),
		})
	}
	// On the raw totals, not the formatted ones: FormatTokens rounds to one
	// decimal, so 1_240_000 and 1_249_999 both render "1.2M" and comparing the
	// strings would order them by text.
	sort.Slice(agents, func(a, b int) bool {
		left, right := totals[agents[a].Key].tokens, totals[agents[b].Key].tokens
		if left != right {
			return left > right
		}
		return agents[a].Key < agents[b].Key
	})
	return agents
}

// historyTotals sums the days the grid actually shows.
//
// Scoped to the rendered window rather than the whole store on purpose: the
// footer sits directly under the grid, and a total covering days the grid does
// not draw would not add up to what is on screen.
func historyTotals(days []HistoryDay, start, today time.Time) (tokens, cost string, count int, label string) {
	from := start.Format("2006-01-02")
	to := today.Format("2006-01-02")

	var sumTokens, sumCost float64
	var earliest string
	for _, day := range days {
		if day.Date < from || day.Date > to {
			continue
		}
		sumTokens += day.Tokens
		sumCost += day.Cost
		if day.Tokens > 0 {
			count++
			if earliest == "" {
				earliest = day.Date
			}
		}
	}

	label = ""
	if earliest != "" {
		if parsed, err := time.Parse("2006-01-02", earliest); err == nil {
			label = parsed.Format("2 Jan 2006") + " – " + today.Format("2 Jan 2006")
		}
	}
	return FormatTokens(sumTokens), FormatCost(sumCost), count, label
}
