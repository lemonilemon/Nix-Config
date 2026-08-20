package collect

// The durable half of the connectivity-policy probe, on the same machinery as
// the speed record (see records.go).
//
// A separate file rather than a second field on the speed record, because the
// two answer different questions and decay at completely different rates: a
// throughput figure is stale in days, where a firewall policy holds for years.
// Merging them would tie a long-lived fact's lifetime to a short-lived one, and
// any future decision to expire speed records would take the policy with it.

// netPolicySnapshotVersion is the on-disk version of the policy record file.
const netPolicySnapshotVersion = 1

// netPolicyRecordLimit caps how many networks are remembered.
//
// The same bound as the speed store, but only because 32 networks is already
// more than anyone revisits -- NOT because the two stores stay in step. They
// prune independently and fill at different rates: this one is written
// automatically on joining, where a speed record only exists if someone pressed
// the button, so a network can easily have a policy record and no speed record.
const netPolicyRecordLimit = 32

// PruneNetPolicyRecords keeps the most recently probed entries.
func PruneNetPolicyRecords(records map[string]NetPolicyRecord, limit int) map[string]NetPolicyRecord {
	return pruneRecords(records, limit)
}

// EncodeNetPolicySnapshot renders the record file.
func EncodeNetPolicySnapshot(records map[string]NetPolicyRecord) (string, error) {
	return encodeRecords(netPolicySnapshotVersion, records)
}

// ParseNetPolicySnapshot decodes the record file.
func ParseNetPolicySnapshot(text string) (map[string]NetPolicyRecord, bool) {
	return parseRecords[NetPolicyRecord](text, netPolicySnapshotVersion)
}

// LoadNetPolicySnapshot reads the record file from path.
func LoadNetPolicySnapshot(path string) (map[string]NetPolicyRecord, bool) {
	return loadRecords[NetPolicyRecord](path, netPolicySnapshotVersion)
}

// SaveNetPolicyRecords writes the in-memory cache to path.
func SaveNetPolicyRecords(path string) error {
	return saveRecords(path, netPolicySnapshotVersion, NetPolicyRecordsSnapshot())
}
