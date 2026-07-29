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

// The per-monitor workspace mapping goes to eww as a plain variable, NOT
// through state.Bar.
//
// That is forced rather than chosen. The golden replay pins 46 ControlHandle
// cases and each holds the whole bar state as a byte-exact JSON string, so any
// new top-level key would break all 46 permanently, and the recording cannot be
// regenerated. `eww update` sidesteps the emit path entirely.
//
// Pushed only on change, for the same reason EmitLoop compares encoded bytes
// before writing: a window move fires several events in a burst, and each push
// is a process spawn.
var (
	monitorViewLock sync.Mutex
	monitorViewLast string
)

// PublishMonitorWorkspaces sends the mapping to eww if it differs from the last
// one sent.
//
// Failures are silent, matching run.Eww: eww may not be listening yet during
// startup, and the bar renders correctly without this variable anyway -- the
// yuck falls back to bar_state.workspace_state whenever monitors is below two.
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

// Update is one batch of assignments into BarState.
//
// Collectors run OUTSIDE the store lock and hand back a closure that applies
// what they found. That split is the point: every collector here shells out,
// and holding the lock across a subprocess would stall the emit loop and every
// control-socket worker behind it.
type Update func(*state.Bar)

// retryDelay is the pause before restarting a watcher whose command died. Long
// enough that a binary missing from PATH cannot become a spin loop.
const retryDelay = 2 * time.Second

// debounceWindow is how long a watcher waits after an event before collecting,
// and how long it then keeps draining. Subsystems emit bursts -- a single
// volume change produces several pactl events -- and collecting per event would
// fork per event.
const debounceWindow = 200 * time.Millisecond

// startCommand is the seam for long-lived streaming children.
//
// Separate from collect.RunText because these are not run-and-capture: the
// watcher reads lines for the life of the daemon. Tests replace it; nothing in
// production does.
var startCommand = func(name string, args ...string) (io.ReadCloser, func(), error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = nil

	// An OPEN stdin, deliberately, and never written to. `bluetoothctl
	// --monitor` exits the moment stdin closes, which turns its watcher into a
	// two-second restart loop; the original passes subprocess.PIPE there for
	// exactly this reason. Harmless for the others, which ignore stdin.
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

// Periodic mirrors watchers.periodic: collect every interval, forever.
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

// PeriodicRefresh mirrors watchers.periodic_refresh: like Periodic, but the
// collector owns writing to the store so it can flag progress and update
// incrementally. Runs once immediately, then every interval.
func PeriodicRefresh(ctx context.Context, store *state.Store, interval time.Duration, refresh func(*state.Store)) {
	for {
		func() {
			// The original wraps this in try/except for a reason: a raise here
			// kills the thread and the AI module never updates again for the
			// rest of the session.
			defer func() { _ = recover() }()
			refresh(store)
		}()
		if !sleep(ctx, interval) {
			return
		}
	}
}

// WatchIdleInhibitor mirrors watchers.watch_idle_inhibitor: re-read the
// inhibitor state on a signal, and at least every 30 seconds so an inhibitor
// toggled outside the bar is picked up.
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

// HyprlandSocketPath mirrors watchers.hyprland_socket_path, reporting ok=false
// where the original returns None.
//
// The XDG_RUNTIME_DIR fallback differs from every other path in this codebase:
// it is /run/user/<uid> rather than /tmp, because that is where Hyprland puts
// its socket regardless of what the environment says.
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

// ApplyMonitorEvent mirrors watchers.apply_monitor_event.
//
// GDK learns about a hotplugged output a beat after Hyprland announces it, so
// the open is retried rather than trusted first time. Each attempt re-checks
// whether someone else has already opened the window -- the startup script, or
// eww's own config reload -- because re-opening an open id makes the bar
// flicker.
func ApplyMonitorEvent(ctx context.Context, action, name string) {
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

// ReconcileBarWindows mirrors watchers.reconcile_bar_windows.
//
// When a monitor connects, eww reloads its whole configuration, which kills and
// respawns this backend -- so the monitoradded event fires while no listener is
// alive and can never be caught. Instead, every start compares live monitors
// against open bar windows and opens whatever is missing.
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
		// Added with the per-monitor mapping. Without these, moving a workspace
		// between screens -- by the bar's own right-click, or by
		// moveworkspacetomonitor from anywhere -- leaves the bars showing the
		// old owner until some unrelated event happens to refresh them.
		// monitoradded/removed matter for the same reason: attaching a screen
		// redistributes workspaces, and the mapping's monitor count is what
		// switches the island between its one-screen and two-screen rendering.
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

// WatchHyprland mirrors watchers.watch_hyprland.
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

// CollectOnce mirrors watchers.collect_once: the seed reading, before any event
// arrives.
//
// Guarded because a panic here runs on the watcher's own goroutine with nothing
// above it -- the goroutine dies and that bar module is frozen for the rest of
// the session, where the identical call inside the loop is retried. The module
// keeps its default until the first event instead.
func CollectOnce(store *state.Store, gather func() Update) {
	defer func() { _ = recover() }()
	if apply := gather(); apply != nil {
		store.Update(apply)
	}
}

// WatchCommand mirrors watchers.watch_command: re-collect whenever a command
// prints a line.
//
// lineFilter drops lines that cannot have changed anything the bar renders.
// Without one, a watcher whose collector shells out to the same subsystem it is
// watching becomes its own event source -- see collect.VolumeEventIsRelevant,
// which was added after `pactl subscribe` was measured driving itself at 1248
// events a minute.
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
//
// The original does this with select() on the child's stdout, which is a latent
// bug -- Python's `for line in stdout` reads through a buffer, so select can
// report the fd unreadable while whole lines sit in userspace, and the drain
// exits early. Reading from a channel fed by the scanner has no such split.
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

// EmitLoop mirrors watchers.emit_loop: write the snapshot, then rewrite it
// every time the state changes.
//
// A closed pipe is a normal end, not an error: eww closes stdout when it stops
// listening, and the daemon should exit quietly rather than log.
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
// long-lived connection, not a request, so it does not go through any of
// package collect's runners.
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

// WatchBluetooth mirrors watchers.watch_bluetooth.
//
// Its own function rather than a WatchCommand call only because it takes no
// line filter and its own seed; the open-stdin requirement `bluetoothctl
// --monitor` has is handled in startCommand, for every watcher.
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
