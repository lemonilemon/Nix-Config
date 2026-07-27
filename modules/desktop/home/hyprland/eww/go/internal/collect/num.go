package collect

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RoundHalfEven is Python's built-in round(): ties go to the even neighbour.
//
// math.Round is NOT this -- it rounds half away from zero, so it disagrees on
// every exact .5, and the bar is full of them (a battery at 62.5%, a quota
// window at 37.5%). Six call sites in the original use int(round(...)), and
// each would be off by one on those inputs with math.Round.
func RoundHalfEven(value float64) int {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	rounded := math.RoundToEven(value)
	return int(rounded)
}

// PercentPart mirrors collectors.percent_part.
func PercentPart(value, total float64) int {
	if value <= 0 || total <= 0 {
		return 0
	}
	return clampInt(RoundHalfEven(value*100/total), 0, 100)
}

// ClampPercent mirrors collectors.clamp_percent.
func ClampPercent(value float64) int {
	return clampInt(RoundHalfEven(value), 0, 100)
}

func clampInt(value, low, high int) int {
	return max(low, min(high, value))
}

// FormatTokens mirrors collectors.format_tokens.
//
// %.1f matches CPython's f-string here: both format the shortest decimal that
// round-trips and break ties to even, so they agree byte for byte. The bare
// branch is int(value), which truncates toward zero rather than rounding.
func FormatTokens(value float64) string {
	switch {
	case value <= 0:
		return "--"
	case value >= 1_000_000:
		return fmt.Sprintf("%.1fM", value/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%.1fK", value/1_000)
	default:
		return strconv.FormatInt(int64(value), 10)
	}
}

// FormatCost mirrors collectors.format_cost.
func FormatCost(value float64) string {
	if value <= 0 {
		return "--"
	}
	return fmt.Sprintf("$%.2f", value)
}

// FormatRemaining mirrors collectors.format_remaining.
//
// The original uses //, which floors; Go's integer division truncates toward
// zero. Identical for the positive values that reach here (<= 0 returns early),
// but spelled with math.Floor so it stays right if that guard ever moves.
func FormatRemaining(seconds float64) string {
	if seconds <= 0 {
		return "0m"
	}
	minutes := int(math.Floor(seconds / 60))
	hours, mins := minutes/60, minutes%60
	if hours != 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// DecimalTimes100HalfUp is Decimal(text) * 100 rounded ROUND_HALF_UP, exactly.
//
// Done on the digits rather than through float64 because that is what the
// original does, and the two disagree: "0.815" is not representable in binary,
// so float64 gives 81.49999999999999 and rounds to 81 where Decimal gives 82.
// wpctl emits two decimals today, where the two agree on all 101 possible
// values -- but "it happens to agree on the inputs we currently see" is not a
// property worth resting on when exactness costs this little.
//
// Reports ok=false when text is not a decimal number, mirroring the caller's
// regex having failed to match.
func DecimalTimes100HalfUp(text string) (int, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	negative := false
	switch text[0] {
	case '+':
		text = text[1:]
	case '-':
		negative = true
		text = text[1:]
	}

	intPart, fracPart, _ := strings.Cut(text, ".")
	if intPart == "" && fracPart == "" {
		return 0, false
	}
	digits := intPart + fracPart
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	if digits == "" {
		return 0, false
	}

	// Multiplying by 100 shifts the point two places right: the first
	// len(fracPart)-2 fractional digits stay fractional.
	shift := len(fracPart) - 2
	if shift < 0 {
		digits += strings.Repeat("0", -shift)
		shift = 0
	}
	whole, frac := digits[:len(digits)-shift], digits[len(digits)-shift:]

	value, err := strconv.Atoi(whole)
	if err != nil {
		return 0, false
	}
	// ROUND_HALF_UP is away from zero on a tie, not toward even.
	if frac != "" && frac[0] >= '5' {
		value++
	}
	if negative {
		value = -value
	}
	return value, true
}
