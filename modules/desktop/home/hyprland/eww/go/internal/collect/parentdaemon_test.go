package collect

import (
	"reflect"
	"testing"
)

// The two argvs the guard exists to tell apart, captured from the duplicate-bar
// incident on 2026-08-20: pid 8234 was the daemon systemd started, pid 8516 was
// the `eww open` that answered a slow reply by forking itself into a second
// one. Both had a backend; only the first one's had any business opening bars.
func TestEwwSubcommandSeparatesDaemonFromClient(t *testing.T) {
	daemon := []string{
		"/nix/store/x766x20cybcwg9zhs83bbq2s9a9dyjrn-eww-0.6.0/bin/eww",
		"--force-wayland", "daemon", "--no-daemonize",
	}
	if got := EwwSubcommand(daemon); got != "daemon" {
		t.Errorf("daemon argv read as %q, want \"daemon\"", got)
	}

	client := []string{
		"eww", "open", "bar",
		"--id", "bar-eDP-1", "--screen", "eDP-1", "--arg", "output=eDP-1",
	}
	if got := EwwSubcommand(client); got != "open" {
		t.Errorf("client argv read as %q, want \"open\"", got)
	}
}

func TestEwwSubcommandSkipsGlobalOptions(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want string
	}{
		// Every global option eww declares, ahead of the subcommand, because
		// `global = true` means they are accepted there.
		{"leading flags", []string{"eww", "--debug", "--force-wayland", "--logs",
			"--no-daemonize", "--restart", "daemon"}, "daemon"},

		// -c takes a following word. A config directory named "daemon" is a
		// silly thing to have and a cheap thing to get wrong.
		{"config value", []string{"eww", "-c", "daemon", "open", "bar"}, "open"},
		{"config long", []string{"eww", "--config", "daemon", "open"}, "open"},
		{"config joined", []string{"eww", "--config=/tmp/x", "daemon"}, "daemon"},

		// Scanning stops at the subcommand, so its own arguments are never
		// candidates -- this one would otherwise read as a daemon.
		{"argument after subcommand", []string{
			"eww", "open", "bar", "--arg", "output=daemon"}, "open"},

		{"no subcommand", []string{"eww"}, ""},
		{"flags only", []string{"eww", "--debug"}, ""},
		{"empty", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EwwSubcommand(tc.argv); got != tc.want {
				t.Errorf("EwwSubcommand(%v) = %q, want %q", tc.argv, got, tc.want)
			}
		})
	}
}

// /proc/<pid>/cmdline terminates the last argument with a NUL as well as
// separating with them, so splitting without the trim leaves a phantom empty
// argument on the end.
func TestArgvFromCmdline(t *testing.T) {
	got := argvFromCmdline([]byte("eww\x00--force-wayland\x00daemon\x00"))
	want := []string{"eww", "--force-wayland", "daemon"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("argv = %#v, want %#v", got, want)
	}

	// A process that has exited reads back as empty rather than as an error.
	if got := argvFromCmdline(nil); got != nil {
		t.Errorf("empty cmdline = %#v, want nil", got)
	}
}
