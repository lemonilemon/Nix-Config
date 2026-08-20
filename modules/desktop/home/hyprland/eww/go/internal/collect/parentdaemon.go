package collect

import (
	"os"
	"strconv"
	"strings"
)

// ParentIsEwwDaemon reports whether the eww process that spawned this backend
// is a daemon, rather than a one-shot client that forked itself into one.
//
// The backend is started by eww's `deflisten`, so its parent IS the eww process
// whose windows it feeds -- pid 8497's parent was the daemon, pid 8639's was an
// `eww open bar --id bar-eDP-1 ...` left over from the duplicate-bar incident on
// 2026-08-20. That is the whole distinction: a client is only supposed to send
// one command and exit, so a client still alive as our parent is a daemon eww
// forked behind systemd's back.
//
// It matters because a backend cannot otherwise tell which daemon it belongs
// to. Every question it asks (`eww active-windows`) and every command it sends
// (`eww open`) travels over $XDG_RUNTIME_DIR/eww-server_<hash>, which exactly
// one daemon owns -- the last one to bind it. So a rogue daemon's backend asks
// the rogue what bars are open, is told none, and opens one. That is how a
// second eww daemon that was asked for nothing but a popup grows a full second
// bar about a second later. See ApplyMonitorEvent, which is the only place that
// opens or closes one.
//
// False when the parent's argv cannot be read at all. Deliberate, and the only
// direction worth defaulting: a missing bar after a monitor hotplug is a
// visible, recoverable annoyance, while an extra bar is the bug being fixed.
// Running eww-bar-backend by hand from a shell lands here too, which is right
// for the same reason -- a hand-run backend has no windows of its own to keep.
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
// Not simply argv[1]. eww declares its global options with `global = true`, so
// they are accepted before the subcommand as well as after it -- the real
// daemon runs as `eww --force-wayland daemon --no-daemonize` -- and -c/--config
// takes a value that must not be read as one. Scanning stops at the subcommand
// so that arguments to it are never considered: `eww open bar --arg
// output=daemon` is an open, not a daemon.
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
