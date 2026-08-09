package control

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"ewwbar/internal/collect"
	"ewwbar/internal/paths"
	"ewwbar/internal/state"
)

// TestMain installs the suite's safety rails.
//
// These tests drive real control handlers, and a handler shells out --
// collect.RunText and RunStatus execute by default. Installing an empty
// fixture replaces every impure seam in package collect before any test runs,
// so nothing here can reach the developer's session. Without it, a `volume
// mute` in a test is a `wpctl set-mute` on the desktop, which is precisely how
// this port muted someone's audio once already.
func TestMain(m *testing.M) {
	restore := collect.InstallFixture(map[string]string{})
	code := m.Run()
	restore()
	os.Exit(code)
}

// isolate points the runtime directory at a per-test temp dir.
//
// EVERY test in this file must call it before anything binds. Listen() removes
// whatever sits at the socket path before binding, so a test that reached the
// real path would delete a running daemon's socket and take over the bar's
// control channel. t.Setenv also fails the build if the test is parallel, which
// is the behaviour we want here rather than a silent race on a shared env var.
func isolate(t *testing.T) string {
	t.Helper()
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	// Assert it took. TempDir is called ONCE and reused: each call returns a
	// fresh directory, so comparing against a second call would compare two
	// different paths and pass for the wrong reason.
	if !strings.HasPrefix(paths.ControlSocket(), runtimeDir) {
		t.Fatalf("socket path %q escaped the temp runtime dir %q",
			paths.ControlSocket(), runtimeDir)
	}
	return runtimeDir
}

func request(t *testing.T, socketPath, payload string) string {
	t.Helper()
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, err := io.WriteString(conn, payload+"\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if unixConn, isUnix := conn.(*net.UnixConn); isUnix {
		_ = unixConn.CloseWrite()
	}
	raw, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return strings.TrimSpace(string(raw))
}

func startServer(t *testing.T) (string, *state.Store) {
	t.Helper()
	_ = isolate(t)

	listener, err := Listen()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	store := state.New()
	go Serve(store, listener)
	return paths.ControlSocket(), store
}

func TestServerAnswersPing(t *testing.T) {
	socketPath, _ := startServer(t)
	got := request(t, socketPath, `{"command":"ping"}`)
	if got != `{"ok":true,"command":"ping"}` {
		t.Fatalf("ping reply = %q", got)
	}
}

func TestServerReportsBadPayloads(t *testing.T) {
	socketPath, _ := startServer(t)

	for _, testCase := range []struct{ name, payload string }{
		{"malformed json", `{"command":`},
		{"not an object", `[1,2,3]`},
		{"unknown command", `{"command":"nope"}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var reply map[string]any
			if err := json.Unmarshal([]byte(request(t, socketPath, testCase.payload)), &reply); err != nil {
				t.Fatalf("reply was not JSON: %v", err)
			}
			if reply["ok"] != false {
				t.Fatalf("expected ok=false, got %v", reply)
			}
			if message, _ := reply["error"].(string); message == "" {
				t.Fatalf("expected an error message, got %v", reply)
			}
		})
	}
}

func TestServerTreatsAnEmptyRequestAsAnEmptyPayload(t *testing.T) {
	// A bare connect-and-close must not be logged as a failure: it is what a
	// probe or a half-open client looks like.
	socketPath, _ := startServer(t)
	got := request(t, socketPath, "")
	if !strings.Contains(got, `"ok":false`) {
		t.Fatalf("empty request should report the unknown-command error, got %q", got)
	}
}

// TestServerServesABurstWithoutDropping checks that 40 concurrent clients all
// get a correct reply.
//
// Deliberately NOT claimed as the regression test for the dropped-scroll bug.
// It passes with a serialised accept loop too -- verified by mutation -- because
// Go's listen backlog comes from somaxconn (4096 here) rather than the
// Python's default, so 40 queued connections are never refused, and a ping is
// answered too fast for serialisation to cost anything. The property that
// actually matters is tested below.
func TestServerServesABurstWithoutDropping(t *testing.T) {
	socketPath, _ := startServer(t)

	const clients = 40
	var wg sync.WaitGroup
	replies := make([]string, clients)
	errs := make([]error, clients)

	for i := range clients {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
			if err != nil {
				errs[index] = err
				return
			}
			defer conn.Close()
			if _, err := io.WriteString(conn, `{"command":"ping"}`+"\n"); err != nil {
				errs[index] = err
				return
			}
			if unixConn, isUnix := conn.(*net.UnixConn); isUnix {
				_ = unixConn.CloseWrite()
			}
			raw, err := io.ReadAll(conn)
			if err != nil {
				errs[index] = err
				return
			}
			replies[index] = strings.TrimSpace(string(raw))
		}(i)
	}
	wg.Wait()

	dropped := 0
	for i := range clients {
		if errs[i] != nil || replies[i] != `{"ok":true,"command":"ping"}` {
			dropped++
		}
	}
	if dropped != 0 {
		t.Fatalf("%d/%d clients dropped (first error: %v, first reply: %q)",
			dropped, clients, errs[0], replies[0])
	}
}

// TestAcceptDoesNotWaitOnASlowHandler is the real regression test.
//
// eww.yuck binds eww-barctl to :onscroll, and a scroll wheel delivers ticks far
// faster than a handler that shells out can be served. If accept() waits on the
// handler, the ticks queue behind it; the Python's serve-then-accept loop let
// the backlog fill and the kernel then refused the rest, which --quiet swallows
// silently. Measured there against 40 clients: 30 dropped.
//
// A burst of instant pings cannot see that. This holds one handler open and
// asserts a second request is still answered promptly.
func TestAcceptDoesNotWaitOnASlowHandler(t *testing.T) {
	socketPath, _ := startServer(t)

	const handlerDelay = 400 * time.Millisecond
	previous := collect.RunStatus
	t.Cleanup(func() { collect.RunStatus = previous })
	collect.RunStatus = func(timeout time.Duration, name string, argv ...string) int {
		for _, arg := range argv {
			if arg == "set-mute" {
				time.Sleep(handlerDelay)
			}
		}
		return 0
	}

	slowDone := make(chan struct{})
	go func() {
		defer close(slowDone)
		request(t, socketPath, `{"command":"volume","action":"mute"}`)
	}()

	// Let the slow handler get as far as its sleep before racing it.
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	got := request(t, socketPath, `{"command":"ping"}`)
	elapsed := time.Since(start)

	if got != `{"ok":true,"command":"ping"}` {
		t.Fatalf("ping reply = %q", got)
	}
	if elapsed > handlerDelay/2 {
		t.Fatalf("ping waited %v behind the slow handler; accept is not concurrent", elapsed)
	}
	<-slowDone
}

func TestListenReplacesAStaleSocket(t *testing.T) {
	_ = isolate(t)

	first, err := Listen()
	if err != nil {
		t.Fatalf("first listen: %v", err)
	}
	// Leak it deliberately: a killed daemon leaves the socket file behind, and
	// binding over it must work rather than failing with EADDRINUSE.
	first.(*net.UnixListener).SetUnlinkOnClose(false)
	_ = first.Close()

	second, err := Listen()
	if err != nil {
		t.Fatalf("second listen over a stale socket: %v", err)
	}
	_ = second.Close()
}

func TestWriteBackendPidfileOnlyRemovesItsOwn(t *testing.T) {
	_ = isolate(t)

	cleanup := WriteBackendPidfile()
	pidfile := paths.BackendPidfile()
	if _, err := os.ReadFile(pidfile); err != nil {
		t.Fatalf("pidfile not written: %v", err)
	}

	// A newer daemon has taken over the pidfile; this one's cleanup must leave
	// it alone rather than deleting the live daemon's.
	if err := os.WriteFile(pidfile, []byte("999999"), 0o644); err != nil {
		t.Fatal(err)
	}
	cleanup()
	if _, err := os.ReadFile(pidfile); err != nil {
		t.Fatal("cleanup deleted a pidfile belonging to another process")
	}
}

// TestPersistWallpaperPick pins the serve-loop half of the login-reveal
// contract. The persist hangs off ServeConnection because the golden replay
// pins Handle and SetWallpaper byte-for-byte; this test is the ordinary-Go
// coverage that replaces what the gate cannot record. Only a "wallpaper set"
// persists, and what it persists is the reply's current -- the wallpaper awww
// actually reports -- not the requested path.
func TestPersistWallpaperPick(t *testing.T) {
	prevWrite := collect.WriteTextFile
	t.Cleanup(func() { collect.WriteTextFile = prevWrite })
	writes := map[string]string{}
	collect.WriteTextFile = func(path, text string) error {
		writes[path] = text
		return nil
	}

	setReply := ok("wallpaper",
		pair("action", "set"),
		pair("wallpaper", collect.Wallpaper{Current: "/w/current.png"}))

	persistWallpaperPick(map[string]any{"command": "wallpaper", "action": "set"}, setReply)
	if len(writes) != 1 {
		t.Fatalf("want exactly one state write, got %v", writes)
	}
	for _, text := range writes {
		if text != "/w/current.png\n" {
			t.Errorf("persisted %q, want the reply's current plus newline", text)
		}
	}

	writes = map[string]string{}
	persistWallpaperPick(map[string]any{"command": "wallpaper", "action": "rescan"}, setReply)
	persistWallpaperPick(map[string]any{"command": "volume", "action": "set"}, setReply)
	if len(writes) != 0 {
		t.Errorf("a non-set command persisted state: %v", writes)
	}
}
