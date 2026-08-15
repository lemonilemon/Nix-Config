package collect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// The durable layer: the two files the AI subsystem keeps under XDG_STATE_HOME
// so a fresh daemon has something true to say before its collectors finish.
//
// Both are caches of things derivable from elsewhere, which is why every read
// path degrades to "no data" rather than to an error. Losing them costs the
// pruned tail of the history and one probe's worth of quota cards; refusing to
// start because one failed to parse would cost the whole bar.

// quotaSnapshotVersion is the on-disk QuotaSnapshot.Version.
const quotaSnapshotVersion = 1

// QuotaSnapshot is the last known provider cards, with the time they were taken.
//
// SavedAt is what makes restoring them honest. A card carries Updated as a bare
// HH:MM with no date, so a snapshot restored the next morning would otherwise
// render "Updated 14:32" as though it were current. Keeping the epoch lets the
// restore say when it was actually taken.
type QuotaSnapshot struct {
	Version int     `json:"version"`
	SavedAt float64 `json:"saved_at"`
	Quotas  []Quota `json:"quotas"`
}

// EncodeQuotaSnapshot renders the quota cache file.
func EncodeQuotaSnapshot(quotas []Quota, nowEpoch float64) (string, error) {
	if quotas == nil {
		quotas = []Quota{}
	}
	encoded, err := json.MarshalIndent(QuotaSnapshot{
		Version: quotaSnapshotVersion,
		SavedAt: nowOr(nowEpoch),
		Quotas:  quotas,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded) + "\n", nil
}

// ParseQuotaSnapshot decodes the quota cache and relabels every card as
// restored, reporting ok=false when there is nothing usable.
//
// The relabel is the substance of this function rather than an afterthought.
// Windows and their percentages are kept, because a card showing yesterday's
// bars beats a card showing nothing and the live probe overwrites it within a
// minute -- but Status becomes "as of HH:MM" so the popup states plainly that
// this is a memory and not a reading. eww.yuck renders quota.status whenever it
// is not "live", so that string lands on screen with no widget change.
//
// Cards that were never live are dropped back to their blank template: a
// remembered "not logged in" or "unavailable" is a stale error, and re-showing
// it at boot would blame the user for a condition that may have cleared.
func ParseQuotaSnapshot(text string) ([]Quota, bool) {
	var snapshot QuotaSnapshot
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, false
	}
	if snapshot.Version > quotaSnapshotVersion || len(snapshot.Quotas) == 0 {
		return nil, false
	}

	asOf := "as of " + restoredClock(snapshot.SavedAt, 0)
	restored := make([]Quota, 0, len(snapshot.Quotas))
	anyLive := false
	for _, card := range snapshot.Quotas {
		if card.Status != "live" {
			restored = append(restored, QuotaDefault(card.Key, card.Name, "waiting"))
			continue
		}
		anyLive = true
		card.Status = asOf
		restored = append(restored, card)
	}
	if !anyLive {
		return nil, false
	}
	return restored, true
}

// restoredClock renders when a snapshot was taken, compactly enough for the
// quota card's status label.
//
// Not FormatClockTime, which always spells the full "2006-01-02 15:04". That
// label sits halign=end in a header row beside the provider name and a plan
// chip inside a 380 px popup, and twenty-one characters there would push the
// row into an ellipsis.
//
// Same day gives the bare time, which is what a snapshot minutes old wants.
// Any older day gives the date instead and drops the time, because for a
// snapshot from last week the minute is noise and the day is the whole point.
// Both forms are unambiguous, which is the property that matters: the reason
// this string exists is that Quota.Updated is a bare HH:MM that cannot say
// whether it means today.
func restoredClock(savedEpoch, nowEpoch float64) string {
	saved := localTime(savedEpoch)
	now := localTime(nowOr(nowEpoch))

	if saved.Year() == now.Year() && saved.YearDay() == now.YearDay() {
		return saved.Format("15:04")
	}
	return saved.Format("Jan 2")
}

// LoadQuotaSnapshot reads the quota cache from path.
func LoadQuotaSnapshot(path string) ([]Quota, bool) {
	if path == "" {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return ParseQuotaSnapshot(string(data))
}

// SaveQuotaSnapshot writes the quota cache to path.
//
// Only fully live sets are written. A partial save -- one live card beside two
// that failed -- would be restored next boot as a card set that never existed
// together, and the cheaper option is to keep the last good one until the next
// clean probe.
func SaveQuotaSnapshot(path string, quotas []Quota, nowEpoch float64) error {
	if path == "" || len(quotas) == 0 {
		return nil
	}
	for _, card := range quotas {
		if card.Status != "live" {
			return nil
		}
	}
	text, err := EncodeQuotaSnapshot(quotas, nowEpoch)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, text)
}

// LoadHistory reads the durable daily rollup from path.
func LoadHistory(path string) []HistoryDay {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return ParseHistoryStore(string(data))
}

// SaveHistory writes the durable daily rollup to path.
func SaveHistory(path string, days []HistoryDay) error {
	if path == "" || len(days) == 0 {
		// Never truncate the file to an empty history. The one way to reach
		// this with no days is a refresh that found nothing, which is exactly
		// when the stored copy is the only copy.
		return nil
	}
	text, err := EncodeHistoryStore(days)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, text)
}

// writeFileAtomic writes text to path via a temporary file and a rename.
//
// Not the WriteTextFile seam, and the difference matters here: that seam calls
// os.WriteFile, which truncates before it writes, so a crash or a full disk
// mid-write leaves a truncated file. Everything else the bar writes -- the
// pidfile, the display mode -- is regenerated within seconds of being lost.
// The history file is not: its whole purpose is holding days no longer
// derivable from anywhere, and a torn write destroys them permanently.
//
// rename(2) within a directory is atomic, so a reader sees the old file or the
// new one and never a partial. The temp file is created in the destination
// directory rather than /tmp because rename cannot cross filesystems.
func writeFileAtomic(path, text string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	// Best effort on the error path: the rename below is what makes the write
	// visible, so anything that fails before it must leave nothing behind.
	defer func() {
		temp.Close()
		os.Remove(tempName)
	}()

	if _, err := temp.WriteString(text); err != nil {
		return err
	}
	// Durability, not just visibility. Without the sync a rename can land in
	// the directory before the data reaches the disk, and a power loss then
	// leaves an empty file where a valid one used to be -- strictly worse than
	// the torn write this function exists to prevent.
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempName, 0o644); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}
