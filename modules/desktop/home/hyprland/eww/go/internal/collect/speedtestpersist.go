package collect

// The durable half of the speed test: one file under XDG_STATE_HOME holding
// what each network measured, keyed by NetworkManager connection UUID.
//
// Everything structural lives in records.go, shared with the connectivity-policy
// store next door. What stays here is what is specific to this record: its
// shape, its version, and how many networks are worth remembering.

// speedtestSnapshotVersion is the on-disk version of the speed record file.
const speedtestSnapshotVersion = 1

// speedtestRecordLimit caps how many networks are remembered.
//
// There is no expiry to go with it, deliberately. A three-month-old measurement
// of a home connection is still roughly true and the card's age label already
// carries the uncertainty, so expiring records could only ever throw away the
// one number there is. What does need bounding is the count: a laptop that joins
// a new network every week would otherwise grow this file forever.
const speedtestRecordLimit = 32

// SpeedtestRecord is one network's last measurement.
//
// Name is carried alongside the numbers so the file can be diagnosed at a
// glance -- a bare UUID says nothing without cross-referencing nmcli. Nothing
// reads it back into the card: the popup shows the connection name from live
// nmcli output, and a remembered name would go stale the moment the profile was
// renamed.
type SpeedtestRecord struct {
	Name       string  `json:"name"`
	Down       float64 `json:"down_mbps"`
	Up         float64 `json:"up_mbps"`
	Latency    float64 `json:"latency_ms"`
	MeasuredAt float64 `json:"measured_at"`
}

// PruneSpeedtestRecords keeps the most recently measured entries.
func PruneSpeedtestRecords(records map[string]SpeedtestRecord, limit int) map[string]SpeedtestRecord {
	return pruneRecords(records, limit)
}

// EncodeSpeedtestSnapshot renders the record file.
func EncodeSpeedtestSnapshot(records map[string]SpeedtestRecord) (string, error) {
	return encodeRecords(speedtestSnapshotVersion, records)
}

// ParseSpeedtestSnapshot decodes the record file.
func ParseSpeedtestSnapshot(text string) (map[string]SpeedtestRecord, bool) {
	return parseRecords[SpeedtestRecord](text, speedtestSnapshotVersion)
}

// LoadSpeedtestSnapshot reads the record file from path.
func LoadSpeedtestSnapshot(path string) (map[string]SpeedtestRecord, bool) {
	return loadRecords[SpeedtestRecord](path, speedtestSnapshotVersion)
}

// SaveSpeedtestRecords writes the in-memory cache to path.
//
// Reads the cache itself rather than taking records, so the one caller -- the
// control handler, after a run finishes -- cannot accidentally persist a stale
// copy it captured before the run.
func SaveSpeedtestRecords(path string) error {
	return saveRecords(path, speedtestSnapshotVersion, SpeedtestRecordsSnapshot())
}
