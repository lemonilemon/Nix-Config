package collect

import (
	"os"
	"strconv"
	"strings"
)

// ParentIsEwwDaemon reports whether the eww process that spawned this backend is a
// daemon, rather than a one-shot client that forked itself into one.
//
// The backend is started by eww's `deflisten`, so its parent IS the eww process
// whose windows it feeds. A client is only supposed to send one command and exit,
// so a client still alive as our parent is a daemon eww forked behind systemd's back.
//
// It matters because every question the backend asks and every command it sends
// travels over $XDG_RUNTIME_DIR/eww-server_<hash>, which exactly one daemon owns.
// So a rogue daemon's backend asks the rogue what bars are open, is told none, and
// opens one. See ApplyMonitorEvent, the only place that opens or closes one.
//
// False when the parent's argv cannot be read at all, which is the only direction
// worth defaulting: a missing bar after a hotplug is recoverable, an extra bar is
// the bug being fixed.
func ParentIsEwwDaemon() bool {
	cmdline, err := os.ReadFile("/proc/" + strconv.Itoa(os.Getppid()) + "/cmdline")
	if err != nil {
		return false
	}
	return EwwSubcommand(argvFromCmdline(cmdline)) == "daemon"
}

// argvFromCmdline splits /proc/<pid>/cmdline, which is NUL-separated and
// NUL-terminated, so a naive Split leaves a phantom empty final argument.
func argvFromCmdline(cmdline []byte) []string {
	trimmed := strings.TrimRight(string(cmdline), "\x00")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\x00")
}

// EwwSubcommand picks the subcommand out of an eww argv, or "" if it has none.
//
// Not simply argv[1]: eww declares its global options with `global = true`, so they
// are accepted before the subcommand as well as after, and -c/--config takes a value
// that must not be read as one. Scanning stops at the subcommand, so `eww open bar
// --arg output=daemon` is an open, not a daemon.
func EwwSubcommand(argv []string) string {
	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		// The one global option that consumes a following word. The --config=X
		// spelling needs no special case; it is caught as a flag below.
		if arg == "-c" || arg == "--config" {
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg
	}
	return ""
}
