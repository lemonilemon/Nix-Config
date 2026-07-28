package collect

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// InstallFixture replaces every impure seam in this package with a lookup into
// a flat string map, and returns a function restoring the originals.
//
// It lives here rather than in the diffgen command because the seams it
// overwrites are unexported-by-convention package state, and because the
// key encoding is a contract the Python side of the equivalence gate has to
// match exactly -- putting it beside the seams keeps the two from drifting.
//
// Keys, all \x1f-separated so a key can never collide with an argument:
//
//	<name>\x1f<arg>\x1f<arg>...   the stdout of that command
//	file\x1f<path>                the contents of that file
//	now                           the wall clock, as float seconds
//	http.status                   the HTTP status to answer with
//	http.body                     the HTTP body to answer with
//	http.error                    non-empty for a transport failure
//	env.<NAME>                    an environment variable
//
// A command or file with no key reads as absent, which is what a missing
// binary or unreadable file produces in production.
//
// NOT concurrency-safe, and does not need to be: diffgen answers one call at a
// time, and nothing in production reassigns these.
func InstallFixture(fixture map[string]string) func() {
	savedRun, savedRead := RunText, ReadTextFile
	savedHTTP, savedNow := httpGetText, timeNow
	savedEnv := map[string]*string{}

	RunText = func(_ time.Duration, name string, argv ...string) string {
		return fixture[strings.Join(append([]string{name}, argv...), "\x1f")]
	}
	ReadTextFile = func(path string) (string, bool) {
		value, ok := fixture["file\x1f"+path]
		return value, ok
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
		httpGetText, timeNow = savedHTTP, savedNow
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
