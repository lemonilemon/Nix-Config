// Package watch holds the daemon's background loops: the periodic collectors,
// the event-driven watchers, and the emit loop that writes state to stdout.
package watch

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"ewwbar/internal/collect"
	"ewwbar/internal/pyjson"
	"ewwbar/internal/run"
	"ewwbar/internal/state"
)

// The per-monitor workspace mapping goes to eww as a plain variable, NOT through
// state.Bar, and is pushed only on change: a window move fires several events in
// a burst and each push is a process spawn.
var (
	monitorViewLock sync.Mutex
	monitorViewLast string
)

// PublishMonitorWorkspaces sends the mapping to eww if it differs from the last
// one sent. Failures are silent: eww may not be listening yet, and the yuck falls
// back to bar_state.workspace_state whenever monitors is below two.
func PublishMonitorWorkspaces(view collect.MonitorWorkspaces) {
	encoded, err := pyjson.Encode(view, false)
	if err != nil {
		return
	}

	monitorViewLock.Lock()
	unchanged := encoded == monitorViewLast
	if !unchanged {
		monitorViewLast = encoded
	}
	monitorViewLock.Unlock()

	if unchanged {
		return
	}
	run.Eww([]string{"update", "ws_monitors=" + encoded})
}

// The AI usage history grid goes to eww the same way, for a reason specific to
// it: eww destroys and rebuilds every child of a `for` whenever any variable the
// loop expression mentions changes -- it watches the variable, not the field
// path. The grid is 53 columns of 7 cells, so reading it from bar_state would
// rebuild 424 widgets every time the 7-second CPU collector ticked.
var (
	aiHistoryLock sync.Mutex
	aiHistoryLast string
)

// PublishAiHistory sends the grid to eww if it differs from the last one sent.
func PublishAiHistory(history collect.AiHistory) {
	encoded, err := pyjson.Encode(history, false)
	if err != nil {
		return
	}

	aiHistoryLock.Lock()
	unchanged := encoded == aiHistoryLast
	if !unchanged {
		aiHistoryLast = encoded
	}
	aiHistoryLock.Unlock()

	if unchanged {
		return
	}
	run.Eww([]string{"update", "ai_history=" + encoded})
}

// Live throughput takes its own variable for the same `for` reason: it ticks every
// 2 s, and putting that on bar_state would rebuild every `for` reading bar_state
// at that rate, including the notification list and the wallpaper grid.
var (
	netRateLock sync.Mutex
	netRateLast string
)

// PublishNetRate sends the throughput readout to eww if it changed. The
// deduplication is what makes a 2 s ticker affordable: an idle connection reports
// "0.0" every tick forever, so the common case spawns no process at all.
func PublishNetRate(rate collect.NetRate) {
	encoded, err := pyjson.Encode(rate, false)
	if err != nil {
		return
	}

	netRateLock.Lock()
	unchanged := encoded == netRateLast
	if !unchanged {
		netRateLast = encoded
	}
	netRateLock.Unlock()

	if unchanged {
		return
	}
	run.Eww([]string{"update", "net_rate=" + encoded})
}

// Update is one batch of assignments into BarState.
//
// Collectors run OUTSIDE the store lock and hand back a closure that applies what
// they found: every collector shells out, and holding the lock across a subprocess
// would stall the emit loop and every control-socket worker behind it.
type Update func(*state.Bar)

// retryDelay is the pause before restarting a watcher whose command died. Long
// enough that a binary missing from PATH cannot become a spin loop.
const retryDelay = 2 * time.Second

// debounceWindow is how long a watcher waits after an event before collecting, and
// how long it then keeps draining. Subsystems emit bursts, and collecting per
// event would fork per event.
const debounceWindow = 200 * time.Millisecond

// startCommand is the seam for long-lived streaming children: the watcher reads
// lines for the life of the daemon rather than run-and-capture. Tests replace it.
var startCommand = func(name string, args ...string) (io.ReadCloser, func(), error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = nil

	// An OPEN stdin, deliberately, and never written to. `bluetoothctl
	// --monitor` exits the moment stdin closes, which turns its watcher into a
	// two-second restart loop. Harmless for the others, which ignore stdin.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, nil, err
	}
	stop := func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}
	return stdout, stop, nil
}

// Periodic collects every interval, forever.
func Periodic(ctx context.Context, store *state.Store, interval time.Duration, gather func() Update) {
	for {
		if apply := gather(); apply != nil {
			store.Update(apply)
		}
		if !sleep(ctx, interval) {
			return
		}
	}
}

// PeriodicRefresh is Periodic except that the collector owns writing to the store,
// so it can flag progress and update incrementally. Runs once immediately.
func PeriodicRefresh(ctx context.Context, store *state.Store, interval time.Duration, refresh func(*state.Store)) {
	for {
		func() {
			// A panic here would kill the goroutine and the AI module would never
			// update again for the rest of the session.
			defer func() { _ = recover() }()
			refresh(store)
		}()
		if !sleep(ctx, interval) {
			return
		}
	}
}

// WatchIdleInhibitor re-reads the inhibitor state on a signal, and at least every
// 30 seconds so an inhibitor toggled outside the bar is picked up.
func WatchIdleInhibitor(ctx context.Context, store *state.Store, refresh <-chan struct{}) {
	apply := func() {
		idle := collect.IdleInhibitedState()
		display := collect.DisplayState("")
		store.Update(func(bar *state.Bar) {
			bar.IdleInhibited = idle
			bar.Display = display
		})
	}
	apply()
	for {
		select {
		case <-ctx.Done():
			return
		case <-refresh:
		case <-time.After(30 * time.Second):
		}
		apply()
	}
}

// HyprlandSocketPath reports ok=false where there is no socket. Its
// XDG_RUNTIME_DIR fallback is /run/user/<uid> rather than the /tmp every other
// path here uses, because that is where Hyprland puts its socket regardless.
func HyprlandSocketPath() (string, bool) {
	signature := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if signature == "" {
		return "", false
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	return filepath.Join(runtimeDir, "hypr", signature, ".socket2.sock"), true
}

// ApplyMonitorEvent retries the open rather than trusting it first time: GDK
// learns about a hotplugged output a beat after Hyprland announces it. Each
// attempt re-checks whether someone else has already opened the window, because
// re-opening an open id makes the bar flicker.
//
// The only place the backend opens or closes a bar window, which is why the
// ownership guard below lives here.
func ApplyMonitorEvent(ctx context.Context, action, name string) {
	// A backend whose parent eww is a client rather than a daemon is a rogue
	// daemon's backend, and these windows are not its to manage. Opening one
	// turns a stray second daemon into a stray second BAR; closing one would
	// take down a bar belonging to the daemon that does own the socket.
	if !collect.ParentIsEwwDaemon() {
		fmt.Fprintf(os.Stderr, "eww-bar: ignoring monitor %s %s -- this "+
			"backend's eww parent is a client, not the daemon\n", action, name)
		return
	}

	attempts := 1
	if action == "added" {
		attempts = 5
	}
	for range attempts {
		if !sleep(ctx, time.Second) {
			return
		}
		if action == "added" {
			if collect.OpenBarNames(collect.RunText(5*time.Second, "eww", "active-windows"))[name] {
				return
			}
		}
		argv := collect.BarWindowCommand(action, name)
		if collect.RunStatus(10*time.Second, argv[0], argv[1:]...) == 0 {
			return
		}
	}
}

// ReconcileBarWindows compares live monitors against open bar windows at every
// start. When a monitor connects, eww reloads its configuration and respawns this
// backend, so the monitoradded event fires while no listener is alive.
func ReconcileBarWindows(ctx context.Context) {
	missing := collect.MissingBarMonitors(
		collect.RunText(5*time.Second, "hyprctl", "monitors", "-j"),
		collect.RunText(5*time.Second, "eww", "active-windows"),
	)
	for _, name := range missing {
		ApplyMonitorEvent(ctx, "added", name)
	}
}

// hyprlandEventPrefixes are the events that change what the bar renders.
var (
	workspacePrefixes = []string{
		"workspace>>", "focusedmon>>", "openwindow>>", "closewindow>>",
		"movewindow>>", "createworkspace>>", "destroyworkspace>>", "urgent>>",
		// Moving a workspace between screens, and attaching one, both redistribute
		// workspaces; the mapping's monitor count is what switches the island
		// between its one-screen and two-screen rendering.
		"moveworkspace>>", "moveworkspacev2>>",
		"monitoradded>>", "monitorremoved>>",
	}
	activeWindowPrefixes = []string{"activewindow>>", "activewindowv2>>", "closewindow>>"}
)

func hasAnyPrefix(line string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func WatchHyprland(ctx context.Context, store *state.Store) {
	go ReconcileBarWindows(ctx)

	for ctx.Err() == nil {
		workspace, monitors := collect.CollectWorkspaceViews()
		store.Update(func(bar *state.Bar) { bar.WorkspaceState = workspace })
		PublishMonitorWorkspaces(monitors)

		socketPath, ok := HyprlandSocketPath()
		if !ok {
			if !sleep(ctx, retryDelay) {
				return
			}
			continue
		}
		if err := readHyprlandEvents(ctx, store, socketPath); err != nil && !sleep(ctx, retryDelay) {
			return
		}
	}
}

func readHyprlandEvents(ctx context.Context, store *state.Store, socketPath string) error {
	conn, err := dialUnix(socketPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		line := scanner.Text()

		if submap, ok := collect.SubmapFromEvent(line); ok {
			store.Update(func(bar *state.Bar) { bar.Submap = submap })
		}
		if hasAnyPrefix(line, workspacePrefixes) {
			workspace, monitors := collect.CollectWorkspaceViews()
			store.Update(func(bar *state.Bar) { bar.WorkspaceState = workspace })
			PublishMonitorWorkspaces(monitors)
		}
		if hasAnyPrefix(line, activeWindowPrefixes) {
			active := collect.CollectActiveWindow()
			store.Update(func(bar *state.Bar) { bar.ActiveWindow = active })
		}
		if action, name, ok := collect.MonitorEvent(line); ok {
			// Its own goroutine: ApplyMonitorEvent sleeps and retries, and must
			// not stall workspace or window event handling behind it.
			go ApplyMonitorEvent(ctx, action, name)
		}
	}
	return scanner.Err()
}

// CollectOnce is the seed reading, before any event arrives. Guarded because a
// panic here runs on the watcher's own goroutine with nothing above it, and would
// freeze that bar module for the rest of the session.
func CollectOnce(store *state.Store, gather func() Update) {
	defer func() { _ = recover() }()
	if apply := gather(); apply != nil {
		store.Update(apply)
	}
}

// WatchCommand re-collects whenever a command prints a line.
//
// lineFilter drops lines that cannot have changed anything the bar renders.
// Without one, a watcher whose collector shells out to the same subsystem it is
// watching becomes its own event source.
func WatchCommand(
	ctx context.Context,
	store *state.Store,
	lineFilter func(string) bool,
	gather func() Update,
	name string,
	args ...string,
) {
	CollectOnce(store, gather)
	for ctx.Err() == nil {
		streamCommand(ctx, store, lineFilter, gather, name, args...)
		if !sleep(ctx, retryDelay) {
			return
		}
	}
}

func streamCommand(
	ctx context.Context,
	store *state.Store,
	lineFilter func(string) bool,
	gather func() Update,
	name string,
	args ...string,
) {
	stdout, stop, err := startCommand(name, args...)
	if err != nil {
		return
	}
	defer stop()

	lines := make(chan string, 256)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64*1024), 1<<20)
		for scanner.Scan() {
			select {
			case lines <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		line, open := <-lines
		if !open {
			return
		}
		if lineFilter != nil && !lineFilter(line) {
			continue
		}
		drain(ctx, lines)
		if apply := gather(); apply != nil {
			store.Update(apply)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// drain waits out a burst: pause, then keep consuming until the source has been
// quiet for one window.
func drain(ctx context.Context, lines <-chan string) {
	timer := time.NewTimer(debounceWindow)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		return
	}
	for {
		timer.Reset(debounceWindow)
		select {
		case _, open := <-lines:
			if !open {
				return
			}
		case <-timer.C:
			return
		case <-ctx.Done():
			return
		}
	}
}

// EmitLoop writes the snapshot, then rewrites it every time the state changes. A
// closed pipe is a normal end, not an error: eww closes stdout when it stops
// listening.
func EmitLoop(ctx context.Context, store *state.Store, out io.Writer) {
	emit := func() bool {
		snapshot, err := store.Snapshot()
		if err != nil {
			return true
		}
		if _, err := io.WriteString(out, snapshot+"\n"); err != nil {
			return !errors.Is(err, io.ErrClosedPipe) && !isEPIPE(err)
		}
		return true
	}

	if !emit() {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-store.Changed():
			if !emit() {
				return
			}
		}
	}
}

// sleep waits, reporting false if the context was cancelled first.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// dialUnix is a seam alongside startCommand: the Hyprland event socket is a
// long-lived connection, not a request.
var dialUnix = func(path string) (io.ReadCloser, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// isEPIPE reports a write to a closed pipe, which Go surfaces as a syscall
// error rather than io.ErrClosedPipe when the far end is a process.
func isEPIPE(err error) bool { return errors.Is(err, syscall.EPIPE) }

// WatchBluetooth is its own function rather than a WatchCommand call because it
// takes no line filter and its own seed.
func WatchBluetooth(ctx context.Context, store *state.Store) {
	gather := func() Update {
		bluetooth := collect.CollectBluetooth()
		return func(bar *state.Bar) { bar.Bluetooth = bluetooth }
	}
	CollectOnce(store, gather)
	for ctx.Err() == nil {
		streamCommand(ctx, store, nil, gather, "bluetoothctl", "--monitor")
		if !sleep(ctx, retryDelay) {
			return
		}
	}
}
