// Command diffgen answers one JSON-encoded call per input line.
//
// It exists so the ported collectors can be checked against the Python they
// came from on the same inputs: tests/eww_bar_backend/test_go_equivalence.py
// feeds both sides the same case file and diffs the answers. Table tests pin
// the cases someone thought of; this pins the ones nobody did.
//
// Protocol, one JSON object per line in, one per line out:
//
//	{"fn": "TruncateText", "args": ["hello", 4]}  ->  {"ok": true, "value": "h..."}
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"ewwbar/internal/collect"
)

type call struct {
	Fn   string            `json:"fn"`
	Args []json.RawMessage `json:"args"`
}

func main() {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 1<<20)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for in.Scan() {
		line := in.Bytes()
		if len(line) == 0 {
			continue
		}
		var c call
		if err := json.Unmarshal(line, &c); err != nil {
			emit(out, nil, err)
			continue
		}
		value, err := dispatch(c)
		emit(out, value, err)
	}
}

func emit(out *bufio.Writer, value any, err error) {
	result := map[string]any{"ok": err == nil}
	if err != nil {
		result["error"] = err.Error()
	} else {
		result["value"] = value
	}
	encoded, _ := json.Marshal(result)
	out.Write(encoded)
	out.WriteByte('\n')
}

func dispatch(c call) (any, error) {
	switch c.Fn {
	case "TruncateText":
		var text string
		var maxLen int
		if err := args(c, &text, &maxLen); err != nil {
			return nil, err
		}
		return collect.TruncateText(text, maxLen), nil

	case "RoundHalfEven":
		var value float64
		if err := args(c, &value); err != nil {
			return nil, err
		}
		return collect.RoundHalfEven(value), nil

	case "PercentPart":
		var value, total float64
		if err := args(c, &value, &total); err != nil {
			return nil, err
		}
		return collect.PercentPart(value, total), nil

	case "ClampPercent":
		var value float64
		if err := args(c, &value); err != nil {
			return nil, err
		}
		return collect.ClampPercent(value), nil

	case "FormatTokens":
		var value float64
		if err := args(c, &value); err != nil {
			return nil, err
		}
		return collect.FormatTokens(value), nil

	case "FormatCost":
		var value float64
		if err := args(c, &value); err != nil {
			return nil, err
		}
		return collect.FormatCost(value), nil

	case "FormatRemaining":
		var seconds float64
		if err := args(c, &seconds); err != nil {
			return nil, err
		}
		return collect.FormatRemaining(seconds), nil

	case "VolumeLabelFromText":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.VolumeLabelFromText(text), nil

	case "VolumeStateFromText":
		var text string
		if err := args(c, &text); err != nil {
			return nil, err
		}
		return collect.VolumeStateFromText(text, nil), nil

	case "VolumeEventIsRelevant":
		var line string
		if err := args(c, &line); err != nil {
			return nil, err
		}
		return collect.VolumeEventIsRelevant(line), nil
	}
	return nil, fmt.Errorf("unknown fn: %s", c.Fn)
}

func args(c call, targets ...any) error {
	if len(c.Args) != len(targets) {
		return fmt.Errorf("%s: want %d args, got %d", c.Fn, len(targets), len(c.Args))
	}
	for i, target := range targets {
		if err := json.Unmarshal(c.Args[i], target); err != nil {
			return fmt.Errorf("%s arg %d: %w", c.Fn, i, err)
		}
	}
	return nil
}
