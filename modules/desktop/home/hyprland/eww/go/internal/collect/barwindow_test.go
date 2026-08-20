package collect

import (
	"slices"
	"testing"
)

// The command that opens a bar must be incapable of forking a second eww
// daemon, and that is what --no-daemonize buys.
//
// eww's client answers a failed server action by starting a daemon of its own,
// gated on `action.can_start_daemon() && !opts.no_daemonize`. The failure that
// fires it is not "no daemon is running": the client gives the daemon ~100 ms
// to reply, and building this bar's widget tree can take longer on a loaded
// machine -- while the daemon goes on to open the window anyway. So a monitor
// hotplugged at a busy moment could leave two eww daemons sharing one IPC
// socket, each with its own bar and its own eww-bar-backend, one of them
// invisible to systemd. The flag is load-bearing.
//
// This is the coverage the golden replay's three `BarWindowCommand("added",
// ...)` cases used to carry; they record the argv from before the flag and are
// skipped there now.
func TestBarWindowCommandCannotForkADaemon(t *testing.T) {
	argv := BarWindowCommand("added", "eDP-1")

	if argv[0] != "eww" || !slices.Contains(argv, "open") {
		t.Fatalf("not an eww open command: %v", argv)
	}
	if !slices.Contains(argv, "--no-daemonize") {
		t.Errorf("open command can fork a rogue daemon: %v", argv)
	}

	// The identity still has to match the open-bars script in
	// eww/default.nix, or a hotplugged monitor gets a window the startup path
	// would not have opened.
	for _, want := range [][2]string{
		{"--id", "bar-eDP-1"},
		{"--screen", "eDP-1"},
		{"--arg", "output=eDP-1"},
	} {
		i := slices.Index(argv, want[0])
		if i < 0 || i+1 >= len(argv) || argv[i+1] != want[1] {
			t.Errorf("%s %s missing from %v", want[0], want[1], argv)
		}
	}
}

// Closing needs no such guard, and should not carry the flag as cargo:
// can_start_daemon() is true for `open` and `open-many` alone, so `eww close`
// against a dead daemon has always just failed.
func TestBarWindowCommandCloseIsUnchanged(t *testing.T) {
	argv := BarWindowCommand("removed", "eDP-1")
	want := []string{"eww", "close", "bar-eDP-1"}
	if !slices.Equal(argv, want) {
		t.Errorf("argv = %v, want %v", argv, want)
	}
}
