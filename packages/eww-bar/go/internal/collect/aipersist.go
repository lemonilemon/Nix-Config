package collect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// The durable layer: the two files the AI subsystem keeps under XDG_STATE_HOME so a
// fresh daemon has something true to say before its collectors finish.
//
// Both are caches of things derivable from elsewhere, so every read path degrades
// to "no data" rather than to an error.

// quotaSnapshotVersion is the on-disk QuotaSnapshot.Version.
const quotaSnapshotVersion = 1

// QuotaSnapshot is the last known provider cards, with the time they were taken.
// SavedAt is what makes restoring them honest: a card carries Updated as a bare
// HH:MM with no date, so a snapshot restored the next morning would otherwise
// render "Updated 14:32" as though it were current.
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

// ParseQuotaSnapshot decodes the quota cache and relabels every card as restored,
// reporting ok=false when there is nothing usable.
//
// The relabel is the substance: windows and percentages are kept, but Status becomes
// "as of HH:MM" so the popup states plainly that this is a memory. eww.yuck renders
// quota.status whenever it is not "live".
//
// Cards that were never live drop back to their blank template: re-showing a
// remembered "not logged in" would blame the user for a condition that may have
// cleared.
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

// restoredClock renders when a snapshot was taken, compactly enough for the quota
// card's status label -- not FormatClockTime's full "2006-01-02 15:04", which would
// push the 380px header row into an ellipsis.
//
// Same day gives the bare time; any older day gives the date and drops the time.
// Both forms are unambiguous, which is the point: Quota.Updated is a bare HH:MM
// that cannot say whether it means today.
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

// SaveQuotaSnapshot writes only fully live sets. A partial save -- one live card
// beside two that failed -- would be restored next boot as a card set that never
// existed together.
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
		// Never truncate the file to an empty history. The one way to reach this
		// with no days is a refresh that found nothing, which is exactly when the
		// stored copy is the only copy.
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
// Not the WriteTextFile seam, which calls os.WriteFile and truncates before it
// writes. Everything else the bar writes is regenerated within seconds of being
// lost; the history file is not, and a torn write destroys days no longer
// derivable from anywhere.
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
