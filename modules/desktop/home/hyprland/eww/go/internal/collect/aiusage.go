package collect

// The ai_usage state: what the bar face shows, plus everything the AI popup
// renders. Assembled from a ccusage report (the periods) and the provider
// quota cards in quota.go.

const glyphAiUsage = "\U000F0674" // U+F0674, the robot glyph on the bar face

// PeriodAgent is one per-agent row inside a period, in emit order.
//
// The original builds this with a sixth key, "sort", holding the raw token
// count, sorts on it and then deletes it. Go sorts on a value it already has,
// so there is no field to delete -- but the emitted key order is the same five.
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

// AiUsage is collectors.ai_usage_state's shape.
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

// AiUsageDefault is common.AI_USAGE_DEFAULT.
//
// A function, not a var: the original deep-copies it at every use, and a shared
// Go value that a caller mutated would corrupt every later reader.
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
