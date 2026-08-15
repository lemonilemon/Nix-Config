package collect

import "testing"

// A trimmed ccusage `daily --json --by-agent` report, keeping only the keys the
// parser reads. Captured from this host rather than invented, and two details
// of the real shape are the ones worth having recorded: the row's own "agent"
// is the string "all" while the real breakdown sits in a sibling "agents"
// ARRAY of objects, and each of those objects repeats the same totalTokens /
// totalCost field names the parent row uses.
const dailyReportJSON = `{
  "daily": [
    {"period":"2026-08-13","agent":"all","totalTokens":54149914,"totalCost":0,
     "metadata":{"agents":["claude"]},
     "agents":[
       {"agent":"claude","totalTokens":54149914,"totalCost":0}
     ]},
    {"period":"2026-08-14","agent":"all","totalTokens":8463812,"totalCost":5.07,
     "metadata":{"agents":["claude","codex"]},
     "agents":[
       {"agent":"codex","totalTokens":463812,"totalCost":0.07},
       {"agent":"claude","totalTokens":8000000,"totalCost":5.00}
     ]}
  ],
  "weekly": [
    {"period":"2026-08-10","agent":"all","totalTokens":62613726,"totalCost":5.07}
  ],
  "monthly": [
    {"period":"2026-08","agent":"all","totalTokens":538202764,"totalCost":732.26}
  ]
}`

// The whole day's total used to be credited to every agent that appeared in it,
// which made the popup's five agent rows sum to 4517M beneath a stated total of
// 2943M. Nothing errored; the numbers just did not add up on screen.
func TestHistoryDaysReadsThePerAgentBreakdown(t *testing.T) {
	days := HistoryDaysFromJSON(dailyReportJSON)

	shared := dayByDate(t, days, "2026-08-14")
	if len(shared.Agents) != 2 {
		t.Fatalf("agents = %v, want two", shared.Agents)
	}

	var sum float64
	for _, agent := range shared.Agents {
		sum += agent.Tokens
	}
	if sum != shared.Tokens {
		t.Errorf("agent tokens sum to %v but the day totals %v", sum, shared.Tokens)
	}

	// Sorted by name, so codex's report-order-first entry lands second.
	if shared.Agents[0].Agent != "claude" || shared.Agents[1].Agent != "codex" {
		t.Errorf("agents = %v, want them sorted by name", shared.Agents)
	}
	if shared.Agents[0].Tokens != 8_000_000 {
		t.Errorf("claude tokens = %v, want its own 8000000 not the day's total",
			shared.Agents[0].Tokens)
	}
}

// The whole point of the detector: a model ccusage cannot price costs exactly
// zero while still burning tokens, and --json mode says nothing about it. This
// is the shape that let claude-opus-5 report $616 of spend as free.
func TestUnpricedModelsNamesModelsThatSpentTokensForFree(t *testing.T) {
	report := `{"daily":[
	  {"period":"2026-08-13","modelBreakdowns":[
	    {"modelName":"claude-opus-5","cost":0,"inputTokens":1644,"outputTokens":11628,
	     "cacheCreationTokens":47099,"cacheReadTokens":1166931},
	    {"modelName":"claude-fable-5","cost":29.49,"inputTokens":2319,"outputTokens":149951}
	  ]}
	]}`

	models := UnpricedModels(report)

	if len(models) != 1 || models[0] != "claude-opus-5" {
		t.Fatalf("models = %v, want [claude-opus-5]", models)
	}
	if got := WarnUnpriced(models); got != "No price for claude-opus-5" {
		t.Errorf("warning = %q", got)
	}
}

// A model that ran no tokens costing nothing is arithmetic, not a missing
// price. Flagging it would make the warning permanent and therefore ignored.
func TestUnpricedModelsIgnoresModelsThatSpentNothing(t *testing.T) {
	report := `{"daily":[{"period":"2026-08-13","modelBreakdowns":[
	  {"modelName":"claude-haiku-4-5","cost":0,"inputTokens":0,"outputTokens":0}
	]}]}`

	if models := UnpricedModels(report); len(models) != 0 {
		t.Errorf("models = %v, want none", models)
	}
}

// The healthy case has to be silent, or the notice becomes furniture.
func TestUnpricedModelsIsSilentWhenEverythingIsPriced(t *testing.T) {
	report := `{"daily":[{"period":"2026-08-13","modelBreakdowns":[
	  {"modelName":"claude-opus-5","cost":43.32,"inputTokens":1644,"outputTokens":11628}
	]}]}`

	if models := UnpricedModels(report); len(models) != 0 {
		t.Errorf("models = %v, want none", models)
	}
	if got := WarnUnpriced(nil); got != "" {
		t.Errorf("warning = %q, want empty", got)
	}
}

// "all" is the row's own total wearing an agent label. Counting it beside the
// real agents would double every figure in the summary.
func TestHistoryDaysSkipsTheAllPseudoAgent(t *testing.T) {
	report := `{"daily":[{"period":"2026-08-13","totalTokens":100,
	  "agents":[{"agent":"all","totalTokens":100},{"agent":"claude","totalTokens":100}]}]}`

	days := HistoryDaysFromJSON(report)

	if len(days) != 1 {
		t.Fatalf("want one day, got %v", days)
	}
	if len(days[0].Agents) != 1 || days[0].Agents[0].Agent != "claude" {
		t.Errorf("agents = %v, want claude alone", days[0].Agents)
	}
}

func dayByDate(t *testing.T, days []HistoryDay, date string) HistoryDay {
	t.Helper()
	for _, day := range days {
		if day.Date == date {
			return day
		}
	}
	t.Fatalf("no day %s in %v", date, days)
	return HistoryDay{}
}

// The weekly section's period is a bare date too -- "2026-08-10" is a real
// calendar day, not a distinguishable rollup key -- so nothing about the string
// separates a week row from a day row. Only reading the daily section keeps
// them apart, and mixing them would double-count a week into one column.
func TestHistoryDaysReadsOnlyTheDailySection(t *testing.T) {
	days := HistoryDaysFromJSON(dailyReportJSON)

	if len(days) != 2 {
		t.Fatalf("want 2 daily rows, got %d: %v", len(days), days)
	}
	for _, day := range days {
		if day.Date == "2026-08" {
			t.Error("a monthly rollup reached the daily history")
		}
	}
	if got := dayByDate(t, days, "2026-08-13").Tokens; got != 54149914 {
		t.Errorf("tokens = %v, want 54149914", got)
	}
}

// "2026-08" parses under several layouts and would sort between "2026-08-01"
// and "2026-08-02" as a string, landing a whole month in one cell.
func TestHistoryDaysRejectsNonCalendarPeriods(t *testing.T) {
	for _, period := range []string{"2026-08", "2026-8-3", "2026", "", "today"} {
		report := `{"daily":[{"period":"` + period + `","totalTokens":1}]}`
		if days := HistoryDaysFromJSON(report); len(days) != 0 {
			t.Errorf("period %q was accepted as a day: %v", period, days)
		}
	}
}

// The entire reason the file exists. ccusage reports what the transcripts still
// hold; once an agent prunes them the day vanishes from every future report,
// and if a merge let that removal through the store would decay to exactly the
// window it was built to outlive.
func TestMergeKeepsDaysFreshNoLongerReports(t *testing.T) {
	stored := []HistoryDay{
		{Date: "2025-11-21", Tokens: 1_000_000, Cost: 1.50, Agents: []HistoryAgentDay{{Agent: "gemini", Tokens: 1_000_000, Cost: 1.50}}},
		{Date: "2026-08-14", Tokens: 8_463_812, Cost: 5.07, Agents: []HistoryAgentDay{{Agent: "claude", Tokens: 8_463_812, Cost: 5.07}}},
	}
	fresh := []HistoryDay{
		{Date: "2026-08-14", Tokens: 8_463_812, Cost: 5.07, Agents: []HistoryAgentDay{{Agent: "claude", Tokens: 8_463_812, Cost: 5.07}}},
	}

	merged := MergeHistoryDays(stored, fresh)

	pruned := dayByDate(t, merged, "2025-11-21")
	if pruned.Tokens != 1_000_000 {
		t.Errorf("pruned day lost its tokens: %v", pruned)
	}
}

// Today's row grows all day, so the newer read has to win.
func TestMergePrefersFreshForADayInBoth(t *testing.T) {
	stored := []HistoryDay{{Date: "2026-08-15", Tokens: 1_000, Cost: 1.00}}
	fresh := []HistoryDay{{Date: "2026-08-15", Tokens: 17_735_931, Cost: 22.56}}

	merged := MergeHistoryDays(stored, fresh)

	if len(merged) != 1 {
		t.Fatalf("want the day merged, not duplicated: %v", merged)
	}
	if merged[0].Tokens != 17_735_931 {
		t.Errorf("tokens = %v, want the fresh 17735931", merged[0].Tokens)
	}
}

// A day can keep its tokens and lose its price: ccusage prices an unknown model
// at zero, which is how claude-opus-5 days read $0.00 under --offline while the
// online table says $43.32. Without this guard one offline refresh would
// overwrite the real figure permanently, and nothing would report it.
func TestMergeKeepsStoredCostWhenFreshPricingCollapses(t *testing.T) {
	stored := []HistoryDay{{Date: "2026-08-13", Tokens: 54_149_914, Cost: 43.32}}
	fresh := []HistoryDay{{Date: "2026-08-13", Tokens: 54_149_914, Cost: 0}}

	merged := MergeHistoryDays(stored, fresh)

	if merged[0].Cost != 43.32 {
		t.Errorf("cost = %v, want the stored 43.32 kept", merged[0].Cost)
	}
}

// The guard above must not freeze a price. When the tokens move, the day
// genuinely changed and the new cost -- zero or not -- is the honest one.
func TestMergeTakesFreshCostWhenTokensAlsoChanged(t *testing.T) {
	stored := []HistoryDay{{Date: "2026-08-13", Tokens: 54_149_914, Cost: 43.32}}
	fresh := []HistoryDay{{Date: "2026-08-13", Tokens: 60_000_000, Cost: 0}}

	merged := MergeHistoryDays(stored, fresh)

	if merged[0].Cost != 0 {
		t.Errorf("cost = %v, want the fresh 0 once tokens moved", merged[0].Cost)
	}
}

// A refresh that fails yields no fresh days. Merging that must not be read as
// "every day went away".
func TestMergeWithAnEmptyReportKeepsEverything(t *testing.T) {
	stored := []HistoryDay{
		{Date: "2026-08-13", Tokens: 54_149_914, Cost: 43.32},
		{Date: "2026-08-14", Tokens: 8_463_812, Cost: 5.07},
	}

	if merged := MergeHistoryDays(stored, nil); len(merged) != 2 {
		t.Errorf("an empty report pruned the store: %v", merged)
	}
}

// The store is read at startup before anything has written it, and a half
// written file survives a crash. Both have to degrade to "no history" rather
// than take the bar down.
func TestParseHistoryStoreSurvivesGarbage(t *testing.T) {
	for _, text := range []string{"", "{", "null", `{"version":1,"days":`, "[]"} {
		if days := ParseHistoryStore(text); len(days) != 0 {
			t.Errorf("input %q yielded days: %v", text, days)
		}
	}
}

// A file written by a later version may mean something else by the same fields.
// Reading it as though it did not would silently corrupt the grid; discarding
// it costs the pruned tail and self-heals on the next write.
func TestParseHistoryStoreRejectsAFutureVersion(t *testing.T) {
	text := `{"version":99,"days":[{"date":"2026-08-13","tokens":1,"cost":1}]}`

	if days := ParseHistoryStore(text); len(days) != 0 {
		t.Errorf("a future version was read as current: %v", days)
	}
}

// Encode and parse are the two halves of the only durable state the bar keeps.
// If they disagree the loss is silent and permanent, since each startup rewrites
// what it managed to read.
func TestHistoryStoreRoundTrips(t *testing.T) {
	days := []HistoryDay{
		{Date: "2025-11-21", Tokens: 1_000_000, Cost: 1.5, Agents: []HistoryAgentDay{{Agent: "gemini", Tokens: 1_000_000, Cost: 1.50}}},
		{Date: "2026-08-13", Tokens: 54_149_914, Cost: 43.32, Agents: []HistoryAgentDay{{Agent: "claude", Tokens: 40_000_000, Cost: 40.00}, {Agent: "codex", Tokens: 14_149_914, Cost: 3.32}}},
	}

	encoded, err := EncodeHistoryStore(days)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded := ParseHistoryStore(encoded)
	if len(decoded) != len(days) {
		t.Fatalf("round trip changed the count: %v", decoded)
	}
	for i, want := range days {
		got := decoded[i]
		if got.Date != want.Date || got.Tokens != want.Tokens || got.Cost != want.Cost {
			t.Errorf("day %d round tripped as %+v, want %+v", i, got, want)
		}
		if len(got.Agents) != len(want.Agents) {
			t.Errorf("day %d agents = %v, want %v", i, got.Agents, want.Agents)
		}
	}
}
