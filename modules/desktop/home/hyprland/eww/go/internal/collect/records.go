package collect

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
)

// The shared store behind the per-network caches: what each network measured,
// keyed by NetworkManager connection UUID.
//
// Keyed by connection rather than a single "last result", which is the substance
// of both features -- one slot would show the hotel's answers while sitting at
// home. Every read path degrades to "no data" rather than to an error.

// datedRecord is one per-network fact that can say when it was taken. Both cards
// render an age, and a record that cannot supply one would be presented as though
// it had just been taken.
type datedRecord interface {
	takenAt() float64
}

func (r SpeedtestRecord) takenAt() float64 { return r.MeasuredAt }
func (r NetPolicyRecord) takenAt() float64 { return r.ProbedAt }

// recordSnapshot is the on-disk shape. The json tags are load-bearing: the speed
// file was written by an earlier version that spelled the struct out by hand, and
// it has to keep parsing.
type recordSnapshot[T any] struct {
	Version int          `json:"version"`
	Records map[string]T `json:"records"`
}

// pruneRecords keeps the `limit` most recently taken entries, breaking ties by
// UUID so the same state always writes the same file.
func pruneRecords[T datedRecord](records map[string]T, limit int) map[string]T {
	if limit <= 0 || len(records) <= limit {
		out := make(map[string]T, len(records))
		for uuid, record := range records {
			out[uuid] = record
		}
		return out
	}

	uuids := make([]string, 0, len(records))
	for uuid := range records {
		uuids = append(uuids, uuid)
	}
	sort.Slice(uuids, func(i, j int) bool {
		left, right := records[uuids[i]].takenAt(), records[uuids[j]].takenAt()
		if left != right {
			return left > right
		}
		return uuids[i] < uuids[j]
	})

	out := make(map[string]T, limit)
	for _, uuid := range uuids[:limit] {
		out[uuid] = records[uuid]
	}
	return out
}

// encodeRecords renders a record file.
func encodeRecords[T datedRecord](version int, records map[string]T) (string, error) {
	if records == nil {
		records = map[string]T{}
	}
	encoded, err := json.MarshalIndent(recordSnapshot[T]{
		Version: version,
		Records: records,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded) + "\n", nil
}

// parseRecords decodes a record file, reporting ok=false when there is nothing
// usable. Undated records are dropped rather than kept: one would render as an
// answer from 1970.
func parseRecords[T datedRecord](text string, version int) (map[string]T, bool) {
	var snapshot recordSnapshot[T]
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, false
	}
	// And nothing after it: a valid snapshot followed by garbage -- a torn write,
	// or a half-restored backup -- would otherwise be accepted on the strength of
	// its first document alone.
	if err := decoder.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return nil, false
	}
	// A file written by a newer daemon is not readable by an older one, and
	// guessing at it would be worse than starting empty.
	if snapshot.Version > version || len(snapshot.Records) == 0 {
		return nil, false
	}

	records := make(map[string]T, len(snapshot.Records))
	for uuid, record := range snapshot.Records {
		if uuid == "" || record.takenAt() <= 0 {
			continue
		}
		records[uuid] = record
	}
	if len(records) == 0 {
		return nil, false
	}
	return records, true
}

// loadRecords reads a record file from path.
func loadRecords[T datedRecord](path string, version int) (map[string]T, bool) {
	if path == "" {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return parseRecords[T](string(data), version)
}

// saveRecords writes a record set to path.
//
// An empty set is never written. The only way to reach that is a save before
// anything was ever measured, and truncating there would discard every
// remembered network in order to record nothing -- the same trade SaveHistory
// refuses to make.
func saveRecords[T datedRecord](path string, version int, records map[string]T) error {
	if path == "" || len(records) == 0 {
		return nil
	}
	text, err := encodeRecords(version, records)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, text)
}
