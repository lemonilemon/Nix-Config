package collect

import (
	"encoding/json"
	"strconv"
	"strings"
)

// The AI usage subsystem reads three third-party reports -- ccusage, openusage-cli
// and the Claude usage endpoint -- none of which this repo controls. These helpers
// probe alternative key names on untyped dicts and fall back to zero.

// pyNumber reports the numeric value of a decoded JSON scalar, matching
// Python's isinstance(value, (int, float)) test.
func pyNumber(value any) (float64, bool) {
	switch v := value.(type) {
	case bool:
		// NOT numeric, though Python's bool subclasses int and would satisfy that
		// isinstance check, which is why number_value opens with an explicit bool
		// guard. Owning the rule here keeps it in one place and testable.
		return 0, false
	case json.Number:
		// Every JSON number arrives as json.Number, because the daemon decodes
		// reports with UseNumber. ErrRange is not a rejection: CPython parses 1e999
		// to inf, and ParseFloat returns ±Inf beside that error.
		number, err := strconv.ParseFloat(v.String(), 64)
		if err == nil {
			return number, true
		}
		if numErr, isNumErr := err.(*strconv.NumError); isNumErr && numErr.Err == strconv.ErrRange {
			return number, true
		}
		return 0, false
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}

// NumberValue is the value of the first key holding a number, or a string float()
// accepts, else 0.
func NumberValue(data map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value := data[key]
		if number, ok := pyNumber(value); ok {
			return number
		}
		if text, isText := value.(string); isText {
			if number, ok := PyFloat(text); ok {
				return number
			}
		}
	}
	return 0
}

func ListValue(data map[string]any, keys ...string) []any {
	for _, key := range keys {
		if list, isList := data[key].([]any); isList {
			return list
		}
	}
	return []any{}
}

// TokenValues is never serialised -- it carries five numbers into period_state --
// but the json tags match the Python keys so the gate can compare them directly.
type TokenValues struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
	Cache  float64 `json:"cache"`
	Total  float64 `json:"total"`
	Cost   float64 `json:"cost"`
}

// DailyTokenValues: the key alternatives are not interchangeable spellings. ccusage
// has renamed these across versions and reports both camelCase and snake_case
// depending on the subcommand, so this order is the precedence that decides which
// number wins.
func DailyTokenValues(row map[string]any) TokenValues {
	input := NumberValue(row, "inputTokens", "input_tokens", "input")
	output := NumberValue(row, "outputTokens", "output_tokens", "output")
	cache := NumberValue(row,
		"cacheCreationTokens", "cacheCreationInputTokens",
		"cache_creation_input_tokens", "cache_creation_tokens",
	) + NumberValue(row,
		"cacheReadTokens", "cacheReadInputTokens",
		"cache_read_input_tokens", "cache_read_tokens",
	)

	total := NumberValue(row, "totalTokens", "total_tokens", "tokens")
	if total <= 0 {
		total = input + output + cache
	}
	return TokenValues{
		Input:  input,
		Output: output,
		Cache:  cache,
		Total:  total,
		Cost:   NumberValue(row, "totalCost", "total_cost", "costUSD", "cost"),
	}
}

// AgentsText names which of the three known agents appear in the key list, in a
// fixed order. The match is substring, not equality, because ccusage reports agent
// keys with vendor prefixes ("anthropic.claude") in some versions.
func AgentsText(agents []string) string {
	var seen []string
	for _, name := range [...]string{"claude", "codex", "gemini"} {
		for _, token := range agents {
			if strings.Contains(token, name) {
				seen = append(seen, name)
				break
			}
		}
	}
	if len(seen) == 0 {
		return "--"
	}
	return strings.Join(seen, " "+middleDot+" ")
}

// PeriodAgentKeys: row["metadata"]["agents"] is iterated with Python's protocol, so
// a string there yields its characters rather than raising. A dict would yield its
// keys in CPython and yields nothing here; the sole consumer is AgentsText, which
// is order-insensitive and substring-matched.
func PeriodAgentKeys(row map[string]any) []string {
	keys := []string{} // Python returns [], and nil would encode as null
	if metadata, isMap := row["metadata"].(map[string]any); isMap {
		switch agents := metadata["agents"].(type) {
		case []any:
			for _, value := range agents {
				if name, isText := value.(string); isText {
					keys = append(keys, name)
				}
			}
		case string:
			for _, char := range agents {
				keys = append(keys, string(char))
			}
		}
	}

	for _, entry := range ListValue(row, "agents") {
		fields, isMap := entry.(map[string]any)
		if !isMap {
			continue
		}
		if name, isText := fields["agent"].(string); isText && name != "all" {
			keys = append(keys, name)
		}
	}

	for i, key := range keys {
		keys[i] = PyLower(key)
	}
	return keys
}
