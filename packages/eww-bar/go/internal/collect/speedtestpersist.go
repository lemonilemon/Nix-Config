package collect

// The durable half of the speed test: one file under XDG_STATE_HOME holding what
// each network measured, keyed by NetworkManager connection UUID. Everything
// structural lives in records.go, shared with the policy store next door.

// speedtestSnapshotVersion is the on-disk version of the speed record file.
const speedtestSnapshotVersion = 1

// speedtestRecordLimit caps how many networks are remembered. There is no expiry
// to go with it: a three-month-old measurement is still roughly true and the card's
// age label carries the uncertainty. What needs bounding is the count.
const speedtestRecordLimit = 32

// SpeedtestRecord is one network's last measurement. Name is carried so the file
// can be diagnosed at a glance, and is never read back into the card -- the popup
// shows the live nmcli name, and a remembered one would go stale on a rename.
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

// SaveSpeedtestRecords reads the cache itself rather than taking records, so the
// one caller cannot accidentally persist a stale copy captured before the run.
func SaveSpeedtestRecords(path string) error {
	return saveRecords(path, speedtestSnapshotVersion, SpeedtestRecordsSnapshot())
}
