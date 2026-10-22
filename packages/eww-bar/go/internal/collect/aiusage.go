package collect

// The ai_usage state: what the bar face shows, plus everything the AI popup
// renders. Assembled from a ccusage report (the periods) and the provider
// quota cards in quota.go.

const glyphAiUsage = "\U000F0674" // U+F0674, the robot glyph on the bar face

// PeriodAgent is one per-agent row inside a period, in emit order. The original
// carries a sixth "sort" key holding the raw token count and deletes it after
// sorting; Go sorts on a value it already has, and the emitted keys are the same five.
type PeriodAgent struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Tokens  string `json:"tokens"`
	Cost    string `json:"cost"`
	Percent int    `json:"percent"`
}

// AiPeriod is one of ai_usage.periods' three entries.
type AiPeriod struct {
	Label  string        `json:"label"`
	Range  string        `json:"range"`
	Tokens string        `json:"tokens"`
	Cost   string        `json:"cost"`
	Agents []PeriodAgent `json:"agents"`
}

// AiPeriods is ai_usage.periods.
type AiPeriods struct {
	Today AiPeriod `json:"today"`
	Week  AiPeriod `json:"week"`
	Month AiPeriod `json:"month"`
}

// AiMeta is ai_usage.meta.
type AiMeta struct {
	Pricing    string `json:"pricing"`
	Refresh    string `json:"refresh"`
	Status     string `json:"status"`
	Stale      string `json:"stale"`
	Refreshing string `json:"refreshing"`
}

type AiUsage struct {
	Text    string    `json:"text"`
	Tooltip string    `json:"tooltip"`
	Class   string    `json:"class"`
	Source  string    `json:"source"`
	Updated string    `json:"updated"`
	Periods AiPeriods `json:"periods"`
	Agents  string    `json:"agents"`
	Quotas  []Quota   `json:"quotas"`
	Meta    AiMeta    `json:"meta"`
}

// AiUsageDefault is a function, not a var: a shared Go value that a caller mutated
// would corrupt every later reader.
func AiUsageDefault() AiUsage {
	blankPeriod := func(label string) AiPeriod {
		return AiPeriod{
			Label: label, Range: "", Tokens: "--", Cost: "--",
			Agents: []PeriodAgent{},
		}
	}
	return AiUsage{
		Text:    glyphAiUsage + " --",
		Tooltip: "AI usage data is not available yet",
		Class:   "missing",
		Source:  "missing",
		Updated: "",
		Periods: AiPeriods{
			Today: blankPeriod("Today"),
			Week:  blankPeriod("This week"),
			Month: blankPeriod("This month"),
		},
		Agents: "--",
		Quotas: QuotaDefaults(),
		Meta: AiMeta{
			Pricing: "offline", Refresh: "5m", Status: "waiting",
			Stale: "false", Refreshing: "false",
		},
	}
}
