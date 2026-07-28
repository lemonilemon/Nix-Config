// Command diffgen answers one JSON-encoded call per input line.
//
// The differential gate drives it: tests/eww_bar_backend/test_go_equivalence.py
// feeds both implementations the same cases and diffs the answers. Table tests
// pin the cases someone thought of; this pins the ones nobody did.
//
// The dispatch itself lives in internal/replay, so the golden replay test can
// run the same calls after the Python is gone.
//
// Protocol, one JSON object per line in, one per line out:
//
//	{"fn": "TruncateText", "args": ["hello", 4]}  ->  {"ok": true, "value": "h..."}
package main

import (
	"bufio"
	"encoding/json"
	"os"

	"ewwbar/internal/replay"
)

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
		var c replay.Call
		if err := json.Unmarshal(line, &c); err != nil {
			emit(out, nil, err)
			continue
		}
		value, err := replay.Answer(c)
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
