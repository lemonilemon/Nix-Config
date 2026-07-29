// Command diffgen answers one JSON-encoded call per input line.
//
// It was built for the differential gate: the Python suite fed both
// implementations the same cases and diffed the answers, which pinned the cases
// nobody thought to write a table test for. That gate went away with the Python,
// and what it produced is now testdata/golden.jsonl.gz.
//
// It still ships because the eww-backend flake check drives it for
// DefaultSnapshot, comparing the daemon's starting state against eww.yuck's
// :initial literal, and because it is the cheapest way to ask the collectors a
// one-off question by hand:
//
//	echo '{"fn":"AgentDisplayName","args":["claude"]}' | go run ./internal/collect/diffgen
//
// The dispatch itself lives in internal/replay, shared with the golden test.
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
