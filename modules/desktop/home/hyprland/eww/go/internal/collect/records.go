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
// Two of these exist -- the speed record and the connectivity-policy record --
// and they want exactly the same machinery: a versioned file under
// XDG_STATE_HOME, an atomic write, a bound on how many networks are remembered,
// and a refusal to keep a record that cannot say when it was taken. This file is
// that machinery once, rather than in each of them.
//
// A map keyed by connection rather than a single "last result", and that is the
// substance of both features rather than an implementation detail. One slot
// would show the hotel's answers while sitting at home; keyed by connection, a
// record can only surface on the network it came from.
//
// Every read path degrades to "no data" rather than to an error. Losing either
// file costs a few facts that can be measured again in seconds, and refusing to
// start because one failed to parse would cost the whole bar.

// datedRecord is one per-network fact that can say when it was taken.
//
// The timestamp is what makes a stored answer safe to show: both cards render an
// age, and a record that cannot supply one would be presented as though it had
// just been taken.
type datedRecord interface {
	takenAt() float64
}

func (r SpeedtestRecord) takenAt() float64 { return r.MeasuredAt }
func (r NetPolicyRecord) takenAt() float64 { return r.ProbedAt }

// recordSnapshot is the on-disk shape. The json tags are load-bearing: the
// speed file was written by an earlier version of this code that spelled the
// struct out by hand, and it has to keep parsing.
type recordSnapshot[T any] struct {
	Version int          `json:"version"`
	Records map[string]T `json:"records"`
}

// pruneRecords keeps the `limit` most recently taken entries.
//
// Ties are broken by UUID so the result is deterministic. Without that, a file
// written twice from the same state could differ, which would make the atomic
// write churn for nothing and any future golden test unreproducible.
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
// usable in it.
//
// Undated records are dropped rather than kept, for the reason datedRecord
// gives: a card's whole contract is that every figure on it can say how old it
// is, and a record that cannot would render as an answer from 1970.
func parseRecords[T datedRecord](text string, version int) (map[string]T, bool) {
	var snapshot recordSnapshot[T]
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, false
	}
	// And nothing after it. A file holding a valid snapshot followed by garbage
	// -- a torn write that writeFileAtomic should prevent but a stray editor or
	// a half-restored backup would not -- would otherwise be accepted on the
	// strength of its first document alone.
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
