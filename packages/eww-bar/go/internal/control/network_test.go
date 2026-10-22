package control

import (
	"strings"
	"testing"
	"time"

	"ewwbar/internal/state"
)

// isolateState points paths.StateFile at a temp dir for the duration of a test.
//
// The empty fixture in TestMain replaces every impure seam in package collect but
// does NOT stub the environment, so paths.Speedtest() resolves against the
// developer's real HOME. Today the run returns early and writes nothing, which is an
// accident of the fixture rather than a guarantee.
func isolateState(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
}

// waitIdle blocks until no speed test is in flight. QueueSpeedtest is fire-and-
// forget, so without this t.Setenv would restore XDG_STATE_HOME out from under the
// save that goroutine is about to do, putting the write back on the real path.
func waitIdle(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		speedtestLock.Lock()
		busy := speedtestBusy
		speedtestLock.Unlock()
		if !busy {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("a speed test was still running after 5s")
}

// These replace the two recorded ControlHandle cases that recordsSupersededNetwork
// skips: the recording pinned the `network` command when wifi-toggle was its only
// action, and both things it asserted have legitimately changed.

// The recorded usage error said "network action must be wifi-toggle". It cannot
// keep saying that while speedtest is also valid.
func TestNetworkUsageErrorNamesEveryAction(t *testing.T) {
	store := state.New()

	_, err := Handle(store, map[string]any{"command": "network", "action": "bogus"})
	if err == nil {
		t.Fatal("a bogus network action was accepted")
	}
	for _, action := range []string{"wifi-toggle", "speedtest"} {
		if !strings.Contains(err.Error(), action) {
			t.Errorf("usage error %q does not mention %q", err, action)
		}
	}
}

// A second click while a run is going must be dropped, not queued: the run moves
// about 70 MB and would measure the second run against the first one's congestion.
func TestNetworkSpeedtestDropsAConcurrentRequest(t *testing.T) {
	store := state.New()

	// Hold the busy flag as a run in flight would, rather than starting a real one:
	// under the fixture cfspeedtest returns nothing, so a real run would finish
	// before the second request arrived and the race would not be the one under test.
	speedtestLock.Lock()
	speedtestBusy = true
	speedtestLock.Unlock()
	t.Cleanup(func() {
		speedtestLock.Lock()
		speedtestBusy = false
		speedtestLock.Unlock()
	})

	if QueueSpeedtest(store) {
		t.Error("a second speed test started while one was already running")
	}

	reply, err := Handle(store, map[string]any{"command": "network", "action": "speedtest"})
	if err != nil {
		t.Fatalf("speedtest command errored: %v", err)
	}
	if !replyHas(reply, "status", "already-running") {
		t.Errorf("reply %v does not report the request was dropped", reply)
	}
}

func TestNetworkSpeedtestStarts(t *testing.T) {
	isolateState(t)
	store := state.New()

	reply, err := Handle(store, map[string]any{"command": "network", "action": "speedtest"})
	if err != nil {
		t.Fatalf("speedtest command errored: %v", err)
	}
	t.Cleanup(func() { waitIdle(t) })
	if !replyHas(reply, "status", "started") {
		t.Errorf("reply %v does not report the run started", reply)
	}
	if !replyHas(reply, "action", "speedtest") {
		t.Errorf("reply %v does not echo the action", reply)
	}
}

// The speed card is rebuilt from the connection the state already knows about. A
// card assembled against a different identity would show one network's numbers
// under another's name.
func TestSpeedtestPublishUsesTheStoredIdentity(t *testing.T) {
	isolateState(t)
	store := state.New()
	store.Update(func(bar *state.Bar) {
		bar.Network.ConnUUID = "uuid-a"
		bar.Network.Class = "wifi"
		bar.Network.Connectivity = "portal"
	})

	if _, err := Handle(store, map[string]any{
		"command": "network", "action": "speedtest"}); err != nil {
		t.Fatalf("speedtest command errored: %v", err)
	}
	// Assert after the run settles, not during: the publish callback fires from the
	// goroutine, so reading mid-flight would test whichever change landed first.
	waitIdle(t)

	// cfspeedtest returns nothing under the fixture, so the run files no record.
	speed := store.Get().Network.Speed
	if speed.Down != "—" || speed.Up != "—" {
		t.Errorf("card shows %q/%q for a network with no record: %+v",
			speed.Down, speed.Up, speed)
	}
	if speed.Status == "running" {
		t.Error("card still reports a run in flight after it finished")
	}
}

func replyHas(reply Reply, key string, want any) bool {
	for _, pair := range reply {
		if pair.Key == key {
			return pair.Value == want
		}
	}
	return false
}
