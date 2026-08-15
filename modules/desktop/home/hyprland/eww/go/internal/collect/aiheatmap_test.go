package collect

import (
	"strings"
	"testing"
	"time"
)

// 2026-08-15 is a Saturday, so the grid's last column is complete and today is
// its bottom cell. Chosen deliberately: a mid-week "today" leaves trailing blank
// cells that would make the counts below ambiguous.
const heatNow = 1786795200 // 2026-08-15 12:00:00 UTC

// heatDays is a year of usage with a known first record and a known shape.
func heatDays() []HistoryDay {
	days := []HistoryDay{}
	start := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	for i := range 200 {
		date := start.AddDate(0, 0, i)
		tokens := float64((i%10)+1) * 1_000_000
		if i%7 == 0 {
			tokens = 0 // an idle day inside the recorded range
		}
		days = append(days, HistoryDay{
			Date:   date.Format("2006-01-02"),
			Tokens: tokens,
			Cost:   tokens / 1_000_000,
			Agents: []HistoryAgentDay{{Agent: "claude", Tokens: tokens, Cost: tokens / 1_000_000}},
		})
	}
	return days
}

// The month labels are a separate widget row from the grid, pinned to it only by
// pixel width. If the two disagree the labels slide against the days they name
// and every date on screen is quietly wrong by a column -- which no assertion
// about the grid itself would catch, and which is invisible until someone hovers
// a cell and reads a date from the wrong month.
func TestMonthLabelsSpanExactlyTheGridWidth(t *testing.T) {
	pinClock(t, heatNow)

	history := HeatmapFromDays(heatDays(), heatNow)

	var labelled int
	for _, month := range history.Months {
		labelled += month.Width
	}
	grid := heatmapWeeks*heatCellPitch - heatCellGap

	if labelled != grid {
		t.Errorf("month row is %d px against a %d px grid; labels drift by %d px",
			labelled, grid, (grid-labelled)/2)
	}
}

// eww's nested `for` reads the outer binding directly, so every column must be a
// full seven-cell array. A short column would silently render a stubby week.
func TestGridIsAlwaysFullyRectangular(t *testing.T) {
	pinClock(t, heatNow)

	history := HeatmapFromDays(heatDays(), heatNow)

	if len(history.Columns) != heatmapWeeks {
		t.Fatalf("columns = %d, want %d", len(history.Columns), heatmapWeeks)
	}
	for i, column := range history.Columns {
		if len(column) != 7 {
			t.Errorf("column %d has %d cells, want 7", i, len(column))
		}
	}
}

// "Nothing recorded" and "recorded nothing" are different claims. Rendering the
// months before the first entry as empty wells would assert the machine sat idle
// through a period nobody was watching.
func TestDaysOutsideTheRecordedRangeAreBlank(t *testing.T) {
	pinClock(t, heatNow)

	history := HeatmapFromDays(heatDays(), heatNow)

	// The grid starts 2025-08-17; the first record is 2025-12-01. Column 0 is
	// therefore entirely before any data.
	for i, cell := range history.Columns[0] {
		if cell.Class != "blank" {
			t.Errorf("cell %d of the first column is %q, want blank", i, cell.Class)
		}
		if cell.Tooltip != "" {
			t.Errorf("a blank cell carries the tooltip %q", cell.Tooltip)
		}
	}

	// An idle day inside the range is a real observation and must NOT be blank.
	var idle int
	for _, column := range history.Columns {
		for _, cell := range column {
			if cell.Class == "" && strings.Contains(cell.Tooltip, "no usage") {
				idle++
			}
		}
	}
	if idle == 0 {
		t.Error("no idle day rendered as a level-0 well; they all became blank")
	}
}

// Today is the last cell with data; everything after it is the future.
func TestFutureDaysAreBlank(t *testing.T) {
	pinClock(t, heatNow)

	history := HeatmapFromDays(heatDays(), heatNow)
	last := history.Columns[len(history.Columns)-1]

	// 2026-08-15 is a Saturday, index 6, so the whole column is past-or-today.
	if last[6].Class == "blank" {
		t.Error("today rendered as blank")
	}
}

// Usage on this host spans three orders of magnitude between a light day and a
// heavy one. A fixed token scale would put nearly every working day in one band;
// quantiles are what keep the grid readable.
func TestLevelsSpreadAcrossTheRampRatherThanSaturating(t *testing.T) {
	pinClock(t, heatNow)

	history := HeatmapFromDays(heatDays(), heatNow)

	counts := map[string]int{}
	for _, column := range history.Columns {
		for _, cell := range column {
			counts[cell.Class]++
		}
	}
	for level := 1; level <= heatLevels; level++ {
		class := "level-" + string(rune('0'+level))
		if counts[class] == 0 {
			t.Errorf("no cell reached %s; the ramp collapsed: %v", class, counts)
		}
	}
}

// The agent rows sit directly under the total. If they are scoped differently
// they will not add up to it, and the panel contradicts itself on screen.
func TestAgentRowsReconcileWithTheStatedTotal(t *testing.T) {
	pinClock(t, heatNow)

	// One agent holding every day's tokens, so the sum is checkable exactly.
	history := HeatmapFromDays(heatDays(), heatNow)

	if len(history.Agents) != 1 {
		t.Fatalf("agents = %v, want one", history.Agents)
	}
	if history.Agents[0].Tokens != history.Tokens {
		t.Errorf("the only agent shows %s but the total says %s",
			history.Agents[0].Tokens, history.Tokens)
	}
}

// A first run, a wiped state directory, or a ccusage that returned nothing.
func TestEmptyHistoryYieldsTheDefaultGrid(t *testing.T) {
	history := HeatmapFromDays(nil, heatNow)

	if len(history.Columns) != 0 {
		t.Errorf("columns = %d, want none so the card renders empty", len(history.Columns))
	}
	if history.Status != "waiting" {
		t.Errorf("status = %q, want waiting", history.Status)
	}
	if history.Tokens != "--" {
		t.Errorf("tokens = %q, want --", history.Tokens)
	}
}

// Every cell's tooltip is pushed to eww inside one argv string, which Linux caps
// at 128 KiB (MAX_ARG_STRLEN). 371 cells leave little room for prose.
func TestTheGridStaysWellUnderTheArgvLimit(t *testing.T) {
	pinClock(t, heatNow)

	history := HeatmapFromDays(heatDays(), heatNow)

	var size int
	for _, column := range history.Columns {
		for _, cell := range column {
			size += len(cell.Class) + len(cell.Tooltip) + 24 // JSON overhead per cell
		}
	}
	if size > 64*1024 {
		t.Errorf("the grid encodes to about %d bytes, over half the 128 KiB argv cap", size)
	}
}
