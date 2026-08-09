package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"ewwbar/internal/collect"
	"ewwbar/internal/paths"
	"ewwbar/internal/pyjson"
	"ewwbar/internal/state"
)

// controlWorkers bounds how many handlers run at once.
//
// Enough to absorb a scroll gesture without letting a stuck handler -- every
// one of them shells out -- spawn work without bound. Excess connections wait
// on a parked goroutine rather than being refused by the kernel, which is what
// the Python's ThreadPoolExecutor queue does.
const controlWorkers = 8

// readPayloadLimit caps a single request. Not in the original, which reads
// until a newline with no ceiling; a client that connects and streams without
// ever sending one would grow the buffer until the daemon died.
const readPayloadLimit = 1 << 20

// ReadPayload mirrors control.read_control_payload: read until a newline or
// EOF, decode as UTF-8 replacing anything invalid, strip, and parse.
//
// An empty request is an empty payload rather than an error, which is what
// makes a bare connect-and-close a no-op instead of a logged failure.
func ReadPayload(conn io.Reader) (map[string]any, error) {
	var buf bytes.Buffer
	chunk := make([]byte, 4096)
	for buf.Len() < readPayloadLimit {
		n, err := conn.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
		}
		if err != nil || bytes.ContainsRune(chunk[:n], '\n') {
			break
		}
	}

	text := strings.TrimSpace(replaceInvalidUTF8(buf.String()))
	if text == "" {
		return map[string]any{}, nil
	}

	var payload map[string]any
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// replaceInvalidUTF8 is bytes.decode("utf-8", errors="replace"). Go strings
// hold arbitrary bytes, so invalid sequences survive unless replaced here, and
// they would then reach the JSON decoder as-is.
func replaceInvalidUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var out strings.Builder
	for i, r := range s {
		if r == utf8.RuneError {
			if _, width := utf8.DecodeRuneInString(s[i:]); width == 1 {
				out.WriteRune(utf8.RuneError)
				continue
			}
		}
		out.WriteRune(r)
	}
	return out.String()
}

// WriteResponse mirrors control.write_control_response. A client that has
// already gone away is not an error worth reporting: eww.yuck's handlers fire
// and forget, so a closed pipe here is the normal end of a --quiet call.
func WriteResponse(conn io.Writer, reply Reply) {
	encoded, err := pyjson.Encode(reply, true)
	if err != nil {
		encoded = `{"ok":false,"error":"unencodable response"}`
	}
	_, _ = io.WriteString(conn, encoded+"\n")
}

// ServeConnection mirrors control.serve_control_connection.
func ServeConnection(store *state.Store, conn net.Conn) {
	defer conn.Close()

	reply := func() Reply {
		payload, err := ReadPayload(conn)
		if err != nil {
			// The message differs from CPython's JSONDecodeError text. The
			// client prints whatever arrives and eww.yuck ignores it on the
			// --quiet path, so the shape is what matters, not the wording.
			return ErrorReply(err.Error())
		}
		result, err := Handle(store, payload)
		if err != nil {
			return ErrorReply(err.Error())
		}
		persistWallpaperPick(payload, result)
		return result
	}()

	WriteResponse(conn, reply)
}

// persistWallpaperPick mirrors nothing in CPython; it is post-port behavior
// (the login-reveal contract), which is why it hangs off the serve loop
// instead of Handle or SetWallpaper -- those journals are pinned by the
// golden replay. It persists the reply's wallpaper.current rather than the
// requested path: current is what awww actually reports on screen, so a
// swallowed awww failure re-records the surviving wallpaper instead of one
// that never appeared.
func persistWallpaperPick(payload map[string]any, reply Reply) {
	if payloadString(payload, "command", "") != "wallpaper" ||
		payloadString(payload, "action", "") != "set" {
		return
	}
	for _, entry := range reply {
		if entry.Key != "wallpaper" {
			continue
		}
		if value, isWallpaper := entry.Value.(collect.Wallpaper); isWallpaper {
			collect.PersistCurrentWallpaper(value.Current)
		}
		return
	}
}

// Listen prepares the control socket and returns a listener.
//
// Split out from Serve so a caller -- and a test -- can bind without also
// entering the accept loop.
func Listen() (net.Listener, error) {
	socketPath := paths.ControlSocket()

	// A stale socket from a killed daemon would make bind fail with EADDRINUSE,
	// so it is removed first. This is also why nothing may run this against the
	// live runtime directory: it unlinks whatever is at that path, including a
	// socket a running daemon is serving on.
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to unlink %s: %w", socketPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return nil, fmt.Errorf("unable to create %s: %w", filepath.Dir(socketPath), err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("unable to bind %s: %w", socketPath, err)
	}
	// Owner only. Best effort, as in the original: a socket that cannot be
	// chmodded is still better than no control channel.
	_ = os.Chmod(socketPath, 0o600)
	return listener, nil
}

// Serve mirrors control.control_server: accept forever, hand each connection
// to a bounded pool.
//
// accept() must never wait on a handler. eww.yuck binds eww-barctl to
// :onscroll, and a scroll wheel delivers ticks far faster than a command that
// shells out can be served; a serve-then-accept loop let the backlog fill and
// the kernel refused the rest with EAGAIN, which the client reports as a failed
// command and --quiet swallows entirely. Measured against a 40-client burst
// before that was fixed in the Python: 30 dropped.
//
// The backlog itself is Go's default rather than the original's explicit 128.
// net.Listen reads /proc/sys/net/core/somaxconn, which is 4096 on this host, so
// the depth is not the constraint the pool size is.
func Serve(store *state.Store, listener net.Listener) {
	slots := make(chan struct{}, controlWorkers)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if isClosed(err) {
				return
			}
			fmt.Fprintf(os.Stderr, "eww-bar control server: accept failed: %v\n", err)
			continue
		}
		// The goroutine waits for a slot; Accept does not. That is the whole
		// point of the pool, and reversing it reintroduces the dropped-scroll
		// bug the comment above describes.
		go func(c net.Conn) {
			slots <- struct{}{}
			defer func() { <-slots }()
			ServeConnection(store, c)
		}(conn)
	}
}

func isClosed(err error) bool {
	return errors.Is(err, net.ErrClosed)
}

// WriteBackendPidfile mirrors control.write_backend_pidfile, returning the
// cleanup the caller should run at shutdown.
//
// The cleanup re-reads the file and only removes it if it still holds OUR pid,
// so a daemon that starts while an old one is shutting down does not have its
// pidfile deleted by the corpse.
//
// Go has no atexit, so the caller wires this to its signal handling. The
// original relies on atexit, which does NOT run on SIGTERM -- so in practice
// the Python leaves its pidfile behind when systemd stops it, and this does
// not. That is a deliberate improvement, not a divergence to preserve.
func WriteBackendPidfile() func() {
	pidfile := paths.BackendPidfile()
	pid := strconv.Itoa(os.Getpid())

	if err := os.MkdirAll(filepath.Dir(pidfile), 0o755); err != nil {
		return func() {}
	}
	if err := os.WriteFile(pidfile, []byte(pid), 0o644); err != nil {
		return func() {}
	}

	return func() {
		current, err := os.ReadFile(pidfile)
		if err == nil && strings.TrimSpace(string(current)) == pid {
			_ = os.Remove(pidfile)
		}
	}
}
