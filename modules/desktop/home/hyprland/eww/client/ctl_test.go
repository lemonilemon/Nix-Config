package main

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestControlPayloadFromArgs carries forward the cases that
// tests/eww_bar_backend/test_ctl.py pinned for the Python client, plus the
// per-command shapes that test_volume.py, test_notifications.py,
// test_network.py, test_bluetooth.py and test_wallpaper.py asserted.
//
// The expected strings are the exact bytes the Python client put on the wire
// (compact separators, insertion-ordered keys), not just decode-equal JSON.
func TestControlPayloadFromArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"ping", []string{"ping"}, `{"command":"ping"}`},
		{"volume up", []string{"volume", "up"}, `{"command":"volume","action":"up"}`},
		{"volume set carries value", []string{"volume", "set", "40"}, `{"command":"volume","action":"set","value":"40"}`},
		{"volume sink carries name", []string{"volume", "sink", "bt_headset"}, `{"command":"volume","action":"sink","sink":"bt_headset"}`},
		{"volume set without value", []string{"volume", "set"}, `{"command":"volume","action":"set"}`},
		{"volume mute", []string{"volume", "mute"}, `{"command":"volume","action":"mute"}`},
		{"media", []string{"media", "play-pause"}, `{"command":"media","action":"play-pause"}`},
		{"bluetooth toggle", []string{"bluetooth", "power-toggle"}, `{"command":"bluetooth","action":"power-toggle"}`},
		{"bluetooth disconnect", []string{"bluetooth", "disconnect", "80:99:E7"}, `{"command":"bluetooth","action":"disconnect","mac":"80:99:E7"}`},
		{"network", []string{"network", "wifi-toggle"}, `{"command":"network","action":"wifi-toggle"}`},
		{"idle defaults to toggle", []string{"idle"}, `{"command":"idle","action":"toggle"}`},
		{"idle explicit", []string{"idle", "status"}, `{"command":"idle","action":"status"}`},
		{"display defaults to status", []string{"display"}, `{"command":"display","action":"status"}`},
		{"display explicit", []string{"display", "toggle"}, `{"command":"display","action":"toggle"}`},
		{"ai refresh", []string{"ai", "refresh"}, `{"command":"ai","action":"refresh"}`},
		{"notif clear-all", []string{"notif", "clear-all"}, `{"command":"notif","action":"clear-all"}`},
		{"notif dismiss", []string{"notif", "dismiss", "17"}, `{"command":"notif","action":"dismiss","id":"17"}`},
		{
			"notif clear-group joins multiword app",
			[]string{"notif", "clear-group", "My", "App"},
			`{"command":"notif","action":"clear-group","app":"My App"}`,
		},
		{
			"notif toggle-group joins multiword app",
			[]string{"notif", "toggle-group", "My", "App"},
			`{"command":"notif","action":"toggle-group","app":"My App"}`,
		},
		{"wallpaper rescan", []string{"wallpaper", "rescan"}, `{"command":"wallpaper","action":"rescan"}`},
		{"wallpaper set", []string{"wallpaper", "set", "/tmp/a.png"}, `{"command":"wallpaper","action":"set","path":"/tmp/a.png"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := controlPayloadFromArgs(tc.args)
			if err != nil {
				t.Fatalf("controlPayloadFromArgs(%q) errored: %v", tc.args, err)
			}
			if got.JSON() != tc.want {
				t.Errorf("controlPayloadFromArgs(%q)\n got %s\nwant %s", tc.args, got.JSON(), tc.want)
			}
		})
	}
}

func TestControlPayloadRejections(t *testing.T) {
	rejected := [][]string{
		{},
		{"--help"},
		{"-h"},
		{"help"},
		{"volume"},
		{"media"},
		{"media", "play-pause", "extra"},
		{"bluetooth"},
		{"network"},
		{"ai"},
		{"ai", "refresh", "extra"},
		{"notif"},
		{"wallpaper"},
		{"nonsense"},
	}
	for _, args := range rejected {
		if _, err := controlPayloadFromArgs(args); err == nil {
			t.Errorf("controlPayloadFromArgs(%q) should have been rejected", args)
		}
	}
}

// TestJSONStringMatchesCPython pins the encoding against literals produced by
// json.dumps(s, separators=(",", ":")) — the ensure_ascii and no-HTML-escaping
// behaviour encoding/json would get wrong.
func TestJSONStringMatchesCPython(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", `"plain"`},
		{"", `""`},
		{`quote"inside`, `"quote\"inside"`},
		{`back\slash`, `"back\\slash"`},
		{"tab\there", `"tab\there"`},
		{"nl\nhere", `"nl\nhere"`},
		{"\x01", `"\u0001"`},
		{"\x1f", `"\u001f"`},
		{"\x7f", `"\u007f"`},
		// CPython leaves these literal; encoding/json escapes them to
		// \u003c, \u003e and \u0026.
		{"a<b>c&d", `"a<b>c&d"`},
		{"slash/here", `"slash/here"`},
		// CPython escapes non-ASCII; encoding/json passes UTF-8 through raw.
		{"caf\u00e9", `"caf\u00e9"`},
		{"\u4e2d\u6587", `"\u4e2d\u6587"`},
		// A Nerd Font PUA glyph of the kind the bar labels carry (U+F017, the
		// clock), and an astral-plane rune, which CPython emits as a surrogate
		// pair rather than a single \U escape.
		{"\uf017", `"\uf017"`},
		{"\U0001f600", `"\ud83d\ude00"`},
	}
	for _, tc := range cases {
		if got := jsonString(tc.in); got != tc.want {
			t.Errorf("jsonString(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

// TestRunCtlAgainstAStubDaemon exercises the real socket path end to end and
// asserts the exit-code and stdout contract eww.yuck depends on.
func TestRunCtlAgainstAStubDaemon(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	listener, err := net.Listen("unix", filepath.Join(dir, "eww-backend.sock"))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	received := make(chan string, 4)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			buf := make([]byte, 4096)
			n, _ := conn.Read(buf)
			received <- strings.TrimSpace(string(buf[:n]))
			// Key order here is the daemon's; the client must not reorder it.
			conn.Write([]byte(`{"ok":true,"command":"ping"}` + "\n"))
			conn.Close()
		}
	}()

	stdout, restore := captureStdout(t)
	code := runCtl([]string{"ping"})
	restore()

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if got := <-received; got != `{"command":"ping"}` {
		t.Errorf("daemon received %s, want {\"command\":\"ping\"}", got)
	}
	if got := strings.TrimSpace(stdout()); got != `{"ok":true,"command":"ping"}` {
		t.Errorf("stdout = %q, want the daemon's bytes verbatim", got)
	}
}

func TestRunCtlQuietPrintsNothingButStillReportsFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir) // no socket in it

	stdout, restore := captureStdout(t)
	code := runCtl([]string{"--quiet", "ping"})
	restore()

	if code != 1 {
		t.Errorf("exit code = %d, want 1 when the daemon is unreachable", code)
	}
	if out := stdout(); out != "" {
		t.Errorf("--quiet printed %q, want nothing", out)
	}
}

func TestRunCtlReportsUsageAsAFailedResponse(t *testing.T) {
	stdout, restore := captureStdout(t)
	code := runCtl([]string{"--help"})
	restore()

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout())), &decoded); err != nil {
		t.Fatalf("usage response is not JSON: %v (%q)", err, stdout())
	}
	if decoded["ok"] != false {
		t.Errorf("ok = %v, want false", decoded["ok"])
	}
	if decoded["error"] != controlUsage {
		t.Errorf("error = %v, want the usage string", decoded["error"])
	}
}

func TestControlSocketPathFallsBackToTmp(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	if got := controlSocketPath(); got != "/tmp/eww-backend.sock" {
		t.Errorf("controlSocketPath() = %q, want /tmp/eww-backend.sock", got)
	}
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	if got := controlSocketPath(); got != "/run/user/1000/eww-backend.sock" {
		t.Errorf("controlSocketPath() = %q", got)
	}
}

func captureStdout(t *testing.T) (func() string, func()) {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer

	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				break
			}
		}
		done <- sb.String()
	}()

	var captured string
	restore := func() {
		writer.Close()
		captured = <-done
		os.Stdout = original
	}
	return func() string { return captured }, restore
}
