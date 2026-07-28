// Package ipc is the control-socket client behind eww-barctl.
package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"ewwbar/internal/paths"
	"ewwbar/internal/pyjson"
)

// Usage is kept byte-identical to the Python client's CONTROL_USAGE.
const Usage = "usage: eww-barctl ping | volume up|down|set <0-100>|mute|sink <name> | " +
	"media play-pause|next|previous | " +
	"idle toggle|on|off|status | display normal|external|headless|restore|toggle|status | ai refresh" +
	" | bluetooth power-toggle|disconnect <mac> | network wifi-toggle" +
	" | notif toggle-group <app>|dismiss <id>|clear-group <app>|clear-all|dnd-toggle|mark-seen" +
	" | wallpaper set <path>|rescan"

var errUsage = errors.New(Usage)

func SocketPath() string { return paths.ControlSocket() }

// controlPayloadFromArgs mirrors ctl.control_payload_from_args exactly,
// including which argument counts are accepted and the key order of each
// payload. tests/eww_bar_backend/test_ctl.py pinned this shape; ctl_test.go
// carries the same cases forward.
func ControlPayloadFromArgs(args []string) (pyjson.Payload, error) {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		return nil, errUsage
	}
	switch args[0] {
	case "ping":
		return pyjson.Payload{pyjson.F("command", "ping")}, nil
	case "volume":
		if len(args) >= 2 {
			p := pyjson.Payload{pyjson.F("command", "volume"), pyjson.F("action", args[1])}
			if args[1] == "set" && len(args) >= 3 {
				p = append(p, pyjson.F("value", args[2]))
			} else if args[1] == "sink" && len(args) >= 3 {
				p = append(p, pyjson.F("sink", args[2]))
			}
			return p, nil
		}
	case "media":
		if len(args) == 2 {
			return pyjson.Payload{pyjson.F("command", "media"), pyjson.F("action", args[1])}, nil
		}
	case "bluetooth":
		if len(args) >= 2 {
			p := pyjson.Payload{pyjson.F("command", "bluetooth"), pyjson.F("action", args[1])}
			if args[1] == "disconnect" && len(args) >= 3 {
				p = append(p, pyjson.F("mac", args[2]))
			}
			return p, nil
		}
	case "network":
		if len(args) >= 2 {
			return pyjson.Payload{pyjson.F("command", "network"), pyjson.F("action", args[1])}, nil
		}
	case "idle":
		action := "toggle"
		if len(args) > 1 {
			action = args[1]
		}
		return pyjson.Payload{pyjson.F("command", "idle"), pyjson.F("action", action)}, nil
	case "display":
		action := "status"
		if len(args) > 1 {
			action = args[1]
		}
		return pyjson.Payload{pyjson.F("command", "display"), pyjson.F("action", action)}, nil
	case "ai":
		if len(args) == 2 {
			return pyjson.Payload{pyjson.F("command", "ai"), pyjson.F("action", args[1])}, nil
		}
	case "notif":
		if len(args) >= 2 {
			p := pyjson.Payload{pyjson.F("command", "notif"), pyjson.F("action", args[1])}
			if (args[1] == "toggle-group" || args[1] == "clear-group") && len(args) >= 3 {
				p = append(p, pyjson.F("app", strings.Join(args[2:], " ")))
			} else if args[1] == "dismiss" && len(args) >= 3 {
				p = append(p, pyjson.F("id", args[2]))
			}
			return p, nil
		}
	case "wallpaper":
		if len(args) >= 2 {
			p := pyjson.Payload{pyjson.F("command", "wallpaper"), pyjson.F("action", args[1])}
			if args[1] == "set" && len(args) >= 3 {
				p = append(p, pyjson.F("path", args[2]))
			}
			return p, nil
		}
	}
	return nil, errUsage
}

// sendControlCommand writes the request and returns the daemon's raw reply.
//
// The reply is returned verbatim rather than re-marshalled: the daemon already
// emits compact, ensure_ascii JSON, so echoing its bytes is byte-identical to
// what the Python client printed after its parse/re-dump round trip -- and it
// cannot reorder keys the way encoding/json would.
func sendControlCommand(p pyjson.Payload) (string, error) {
	conn, err := net.Dial("unix", SocketPath())
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if _, err := io.WriteString(conn, p.JSON()+"\n"); err != nil {
		return "", err
	}
	if unixConn, ok := conn.(*net.UnixConn); ok {
		// Mirrors the Python client's shutdown(SHUT_WR): the daemon reads until
		// EOF or newline, and half-closing is what lets it use either.
		if err := unixConn.CloseWrite(); err != nil {
			return "", err
		}
	}
	raw, err := io.ReadAll(conn)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func Run(args []string) int {
	quiet := false
	if len(args) > 0 && (args[0] == "-q" || args[0] == "--quiet") {
		quiet = true
		args = args[1:]
	}

	response, err := func() (string, error) {
		p, err := ControlPayloadFromArgs(args)
		if err != nil {
			return "", err
		}
		return sendControlCommand(p)
	}()

	switch {
	case err != nil:
		// Matches the Python client, which wrapped every failure -- bad usage,
		// no socket, a refused connection -- as a failed response rather than a
		// traceback, because these run from :onclick handlers. The `error`
		// text for I/O failures reads differently from CPython's (Go: "dial
		// unix ...: connect: no such file or directory"); the shape, the exit
		// code and the --quiet silence are what eww.yuck depends on, and all 25
		// of its eww-barctl call sites pass --quiet.
		response = errorResponse(err.Error())
	case !json.Valid([]byte(response)):
		response = errorResponse("invalid backend response")
	}

	if !quiet {
		fmt.Println(response)
	}
	if responseIsOK(response) {
		return 0
	}
	return 1
}

func errorResponse(message string) string {
	return `{"ok":false,"error":` + pyjson.String(message) + `}`
}

func responseIsOK(raw string) bool {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return false
	}
	ok, _ := decoded["ok"].(bool)
	return ok
}
