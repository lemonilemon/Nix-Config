package collect

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// InstallFixture replaces every impure seam in this package with a lookup into
// a flat string map, and returns a function restoring the originals.
//
// It lives here rather than in the diffgen command because the seams it
// overwrites are unexported-by-convention package state, and because the key
// encoding is a wire contract: the golden replay's recorded cases carry these
// exact keys, so changing one silently invalidates the recording.
//
// Keys, all \x1f-separated so a key can never collide with an argument:
//
//	<name>\x1f<arg>\x1f<arg>...   the stdout of that command
//	file\x1f<path>                the contents of that file
//	now                           the wall clock, as float seconds
//	http.status                   the HTTP status to answer with
//	http.body                     the HTTP body to answer with
//	http.error                    non-empty for a transport failure
//	status\x1f<name>\x1f<arg>...   the exit code of that command, default 0
//	dir\x1f<path>                  a directory listing, one entry per line as
//	                              "<name>|<isfile>|<mtime_ns>|<size>"
//	exists\x1f<path>               "1" if the path exists
//	boottime                      CLOCK_BOOTTIME seconds
//	resolve\x1f<path>              what that path resolves to
//	env.<NAME>                    an environment variable
//	dial\x1f<address>              "1" if a TCP connect to it should succeed
//	lookup\x1f<host>               resolved addresses, one per line
//
// A command or file with no key reads as absent, which is what a missing
// binary or unreadable file produces in production.
//
// Installing a fixture is NOT concurrency-safe and does not need to be: diffgen
// answers one call at a time, and nothing in production reassigns these. The
// JOURNAL below is a separate matter and is locked, because a collector running
// under a fixture may now fan out across goroutines.
func InstallFixture(fixture map[string]string) func() {
	savedRun, savedRead := RunText, ReadTextFile
	savedStatus, savedWrite, savedRemove := RunStatus, WriteTextFile, RemoveFile
	savedHTTP, savedNow := httpGetText, timeNow
	savedList, savedResolve, savedExists := ListDir, ResolvePath, FileExists
	savedBoottime := BoottimeSeconds
	savedDial, savedLookup := DialTCP, LookupHost
	savedEnv := map[string]*string{}
	fixtureJournalLock.Lock()
	fixtureJournal = nil
	fixtureJournalLock.Unlock()

	// The network seams default to FAILING rather than to succeeding, unlike
	// RunStatus above. A probe that quietly reached the real internet from `go
	// test` is exactly what this rail exists to prevent, and "everything is
	// blocked" is the safe reading of an unconfigured fixture.
	DialTCP = func(address string, _ time.Duration) bool {
		record("dial\x1f" + address)
		return fixture["dial\x1f"+address] == "1"
	}
	LookupHost = func(host string) ([]string, error) {
		record("lookup\x1f" + host)
		value, ok := fixture["lookup\x1f"+host]
		if !ok || value == "" {
			return nil, fixtureError("no such host: " + host)
		}
		return SplitLines(value), nil
	}

	RunText = func(_ time.Duration, name string, argv ...string) string {
		key := strings.Join(append([]string{name}, argv...), "\x1f")
		record("run\x1f" + key)
		return fixture[key]
	}
	ReadTextFile = func(path string) (string, bool) {
		value, ok := fixture["file\x1f"+path]
		return value, ok
	}
	RunStatus = func(_ time.Duration, name string, argv ...string) int {
		key := strings.Join(append([]string{name}, argv...), "\x1f")
		record("status\x1f" + key)
		// Absent means SUCCESS. The display and inhibitor tests care about the
		// ORDER of the calls, and defaulting to failure would make every
		// fixture carry a key per command just to get past the first one.
		code, err := strconv.Atoi(fixture["status\x1f"+key])
		if err != nil {
			return 0
		}
		return code
	}
	WriteTextFile = func(path, text string) error {
		record("write\x1f" + path + "\x1f" + text)
		return nil
	}
	RemoveFile = func(path string) error {
		record("remove\x1f" + path)
		return nil
	}
	httpGetText = func(string, map[string]string, time.Duration) (int, string, error) {
		if message := fixture["http.error"]; message != "" {
			return 0, "", fixtureError(message)
		}
		status, err := strconv.Atoi(fixture["http.status"])
		if err != nil {
			status = 200
		}
		return status, fixture["http.body"], nil
	}
	ListDir = func(dir string) ([]DirEntryInfo, bool) {
		raw, present := fixture["dir\x1f"+dir]
		if !present {
			return nil, false
		}
		listing := []DirEntryInfo{}
		for _, line := range SplitLines(raw) {
			parts := strings.Split(line, "|")
			if len(parts) != 4 {
				continue
			}
			mtime, _ := strconv.ParseInt(parts[2], 10, 64)
			size, _ := strconv.ParseInt(parts[3], 10, 64)
			listing = append(listing, DirEntryInfo{
				Path:    dir + "/" + parts[0],
				IsFile:  parts[1] == "1",
				MtimeNS: mtime,
				Size:    size,
			})
		}
		return listing, true
	}
	ResolvePath = func(path string) (string, bool) {
		if resolved, present := fixture["resolve\x1f"+path]; present {
			return resolved, true
		}
		// Absent means "resolves to itself", which is what an ordinary
		// non-symlink file does and keeps the fixtures short.
		return path, true
	}
	FileExists = func(path string) bool { return fixture["exists\x1f"+path] == "1" }
	if raw, present := fixture["boottime"]; present {
		if seconds, err := strconv.ParseFloat(raw, 64); err == nil {
			BoottimeSeconds = func() float64 { return seconds }
		}
	}
	if raw, present := fixture["now"]; present {
		if seconds, err := strconv.ParseFloat(raw, 64); err == nil {
			frozen := time.Unix(int64(seconds), int64((seconds-float64(int64(seconds)))*1e9))
			timeNow = func() time.Time { return frozen }
		}
	}
	for key, value := range fixture {
		name, isEnv := strings.CutPrefix(key, "env.")
		if !isEnv {
			continue
		}
		savedEnv[name] = envSnapshot(name)
		setEnv(name, value)
	}

	return func() {
		RunText, ReadTextFile = savedRun, savedRead
		RunStatus, WriteTextFile, RemoveFile = savedStatus, savedWrite, savedRemove
		httpGetText, timeNow = savedHTTP, savedNow
		ListDir, ResolvePath, FileExists = savedList, savedResolve, savedExists
		BoottimeSeconds = savedBoottime
		DialTCP, LookupHost = savedDial, savedLookup
		for name, previous := range savedEnv {
			restoreEnv(name, previous)
		}
	}
}

type fixtureError string

func (e fixtureError) Error() string { return string(e) }

func envSnapshot(name string) *string {
	if value, present := os.LookupEnv(name); present {
		return &value
	}
	return nil
}

func setEnv(name, value string) { _ = os.Setenv(name, value) }

func restoreEnv(name string, previous *string) {
	if previous == nil {
		_ = os.Unsetenv(name)
		return
	}
	_ = os.Setenv(name, *previous)
}

// fixtureJournal records every impure operation InstallFixture intercepted, in
// order. It exists because several of these collectors are defined by their
// SIDE EFFECTS rather than their return value -- set_display_mode's whole
// content is which hyprctl and systemctl calls it makes and in what order, and
// a test that only compared the returned Display would pass with the body
// deleted.
//
// Guarded, unlike the seam variables themselves. Installing a fixture is still a
// single-threaded act, but the collectors running under one are no longer all
// sequential: ProbeNetPolicy fans out to five goroutines that each hit a seam,
// and an unlocked append from those raced under -race. The lock covers the
// journal only -- swapping the seams during a run would still be a mistake.
var (
	fixtureJournalLock sync.Mutex
	fixtureJournal     []string
)

func record(entry string) {
	fixtureJournalLock.Lock()
	defer fixtureJournalLock.Unlock()
	fixtureJournal = append(fixtureJournal, entry)
}

// FixtureJournal returns the operations recorded since InstallFixture ran.
//
// A copy, so a caller reading it while a concurrent collector is still
// recording cannot observe the slice being reallocated underneath.
func FixtureJournal() []string {
	fixtureJournalLock.Lock()
	defer fixtureJournalLock.Unlock()
	out := make([]string, len(fixtureJournal))
	copy(out, fixtureJournal)
	return out
}
