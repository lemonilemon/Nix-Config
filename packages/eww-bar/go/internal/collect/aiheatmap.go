package collect

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// The contribution-graph grid behind the popup's History tab.
//
// Columns is a plain [][]HeatCell rather than a slice of structs with a Cells
// field, and that is load-bearing: eww's `for` body must be exactly one widget and
// a `for` may not be the direct body of another `for`, so the nesting has to be
// for -> box -> for with the inner loop iterating the outer binding DIRECTLY.
//
// The structure reaches eww as its own variable via `eww update`, never through
// state.Bar. Two sufficient reasons: adding a field to AiUsage breaks 112 recorded
// golden cases and one to state.Bar breaks 46, none regenerable; and eww rebuilds
// every child of a `for` when any variable it mentions changes, so a grid fed from
// bar_state would rebuild 424 widgets every 7 seconds.

// heatmapWeeks is how many week-columns the grid renders; historyDays is how far
// back the Year period's NUMBERS reach. Deliberately different: the grid is bounded
// by the window (53 weeks forces ~540px against this bar's 280-340px popups), while
// the totals stay a full year, because shrinking them would redefine "Year".
const (
	heatmapWeeks = 26
	historyDays  = 365
)

// heatLevels is how many non-zero shades the ramp has, matching the .level-N
// classes in eww.scss. Level 0 is the base .ai-heat-cell rule and emits no
// modifier class at all.
const heatLevels = 4

// HeatCell is one day. Class carries the whole visual state and Tooltip the whole
// hover text, so the yuck template stays a single interpolation of each: anything
// conditional would be a simplexpr ternary re-evaluated 371 times per rebuild.
type HeatCell struct {
	Class   string `json:"class"`
	Tooltip string `json:"tooltip"`
}

// HeatMonth is one month label above the grid. Width is a pixel count rather than
// a column span because labels are variable-width text where the columns are fixed
// 5px squares.
type HeatMonth struct {
	Label string `json:"label"`
	Width int    `json:"width"`
}

// AiHistory is the ai_history eww variable.
//
// Period is an AiPeriod, the same shape bar_state.ai_usage.periods uses, so the
// year renders through eww.yuck's existing period_panel widget. Warning lives here
// rather than on AiUsage because that struct is pinned by 112 recorded cases.
type AiHistory struct {
	Columns [][]HeatCell `json:"columns"`
	Months  []HeatMonth  `json:"months"`
	Period  AiPeriod     `json:"period"`
	Days    int          `json:"days"`
	Status  string       `json:"status"`
	Warning string       `json:"warning"`
}

// heatCellPitch and heatCellGap mirror .ai-heat-cell's min-width and the grid
// boxes' :spacing in eww.scss. Duplicated here only to size the month labels;
// the cells themselves are sized entirely by CSS.
const (
	heatCellPitch          = 12 // 10 px cell + 2 px spacing
	heatCellGap            = 2
	heatMonthLabelMinWidth = 24

	// How many agent rows the summary shows before folding the tail into a single
	// "+N more". Five is what fits above the footer at the window's 420px.
	maxHistoryAgentRows = 5
)

// AiHistoryDefault is an empty grid rather than a placeholder-filled one: the yuck
// renders whatever columns exist, so zero columns needs no "no data" branch.
func AiHistoryDefault() AiHistory {
	return AiHistory{
		Columns: [][]HeatCell{},
		Months:  []HeatMonth{},
		Period: AiPeriod{
			Label: "This year", Range: "", Tokens: "--", Cost: "--",
			Agents: []PeriodAgent{},
		},
		Days:    0,
		Status:  "waiting",
		Warning: "",
	}
}

// WarnUnpriced renders the unpriced-model notice, empty when every model that spent
// tokens also has a price. It names the models because the name is the key to add
// to the pricingOverrides table in overlays/default.nix.
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

// heatThresholds returns the upper bound of each of the first heatLevels-1 bands,
// as quantiles over the days that had any usage.
//
// Quantiles rather than a fixed token scale because usage spans three orders of
// magnitude here. Zero days are excluded: including them on a machine used twice a
// week would put the median at zero and push every active day to the top band.
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
		// with index len, level 4 could never be reached.
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
// Columns run oldest to newest, seven cells each, Sunday first.
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

	// The grid ends on the Saturday of the current week, so today's column is last.
	start := today.AddDate(0, 0, -int(today.Weekday())-(heatmapWeeks-1)*7)

	// The numbers reach back further than the graph draws; see historyDays.
	totalsFrom := today.AddDate(0, 0, -historyDays)

	// Days before the first record are not "no usage" but "not recorded", and an
	// empty well would claim the machine sat idle through months it was not watched.
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

		// A month owns the columns whose Sunday falls inside it, which keeps each
		// label at or before its month's first full week rather than a column late.
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
	tokens, cost, count, label, rawTotal := historyTotals(days, totalsFrom, today)
	history.Days = count
	history.Period = AiPeriod{
		Label:  "This year",
		Range:  label,
		Tokens: tokens,
		Cost:   cost,
		Agents: historyAgents(days, totalsFrom, today, rawTotal),
	}
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
// A month is span*pitch wide and only the LAST one loses a gap: the gap lives
// between columns, so subtracting one from every month made the label row 12px
// narrower than the grid and offset every label by 6px.
//
// A one-column leading label and a trailing label under 24px are dropped, because
// edge labels otherwise collide with the card edge. Their width is retained.
func monthRow(labels []string, spans []int) []HeatMonth {
	months := make([]HeatMonth, 0, len(labels))
	for i, label := range labels {
		width := spans[i] * heatCellPitch
		if i == len(labels)-1 {
			width -= heatCellGap
		}
		if (i == 0 && spans[i] < 2) || (i == len(labels)-1 && width < heatMonthLabelMinWidth) {
			label = ""
		}
		months = append(months, HeatMonth{Label: label, Width: width})
	}
	return months
}

// historyAgents totals the window per agent, busiest first. Scoped to the same
// window as historyTotals, so the rows add up to the figure printed beside them.
func historyAgents(days []HistoryDay, start, today time.Time, total float64) []PeriodAgent {
	from, to := start.Format("2006-01-02"), today.Format("2006-01-02")

	type sums struct {
		tokens float64
		cost   float64
	}
	totals := map[string]*sums{}
	for _, day := range days {
		if day.Date < from || day.Date > to {
			continue
		}
		for _, agent := range day.Agents {
			if totals[agent.Agent] == nil {
				totals[agent.Agent] = &sums{}
			}
			totals[agent.Agent].tokens += agent.Tokens
			totals[agent.Agent].cost += agent.Cost
		}
	}

	type row struct {
		key    string
		tokens float64
		cost   float64
	}
	rows := make([]row, 0, len(totals))
	for key, sum := range totals {
		rows = append(rows, row{key: key, tokens: sum.tokens, cost: sum.cost})
	}
	sort.Slice(rows, func(a, b int) bool {
		if rows[a].tokens != rows[b].tokens {
			return rows[a].tokens > rows[b].tokens
		}
		return rows[a].key < rows[b].key
	})

	// Fold the tail into one row once there are more agents than the panel can
	// show. The summary sits in a scroll, so without this the extra rows are simply
	// not on screen and nothing says they exist. Aggregating keeps them adding up.
	if len(rows) > maxHistoryAgentRows {
		var rest row
		for _, r := range rows[maxHistoryAgentRows-1:] {
			rest.tokens += r.tokens
			rest.cost += r.cost
		}
		hidden := len(rows) - (maxHistoryAgentRows - 1)
		rows = append(rows[:maxHistoryAgentRows-1], row{
			key:    fmt.Sprintf("+%d more", hidden),
			tokens: rest.tokens,
			cost:   rest.cost,
		})
	}

	agents := make([]PeriodAgent, 0, len(rows))
	for _, r := range rows {
		name := AgentDisplayName(r.key)
		key := r.key
		if strings.HasPrefix(r.key, "+") {
			// Not an agent id: it must not pick up a provider dot colour, and
			// AgentDisplayName would title-case it into "+2 More".
			name, key = r.key, "more"
		}
		agents = append(agents, PeriodAgent{
			Key:     key,
			Name:    name,
			Tokens:  FormatTokens(r.tokens),
			Cost:    FormatCost(r.cost),
			Percent: PercentPart(r.tokens, total),
		})
	}
	return agents
}

// historyTotals sums the days the grid actually shows: the footer sits directly
// under the grid, and a total covering days it does not draw would not add up.
func historyTotals(days []HistoryDay, start, today time.Time) (tokens, cost string, count int, label string, rawTokens float64) {
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
	return FormatTokens(sumTokens), FormatCost(sumCost), count, label, sumTokens
}
