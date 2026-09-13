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

// controlWorkers bounds how many handlers run at once. Excess connections wait on
// a parked goroutine rather than being refused by the kernel.
const controlWorkers = 8

// readPayloadLimit caps a single request: a client that streams without ever
// sending a newline would otherwise grow the buffer until the daemon died.
const readPayloadLimit = 1 << 20

// ReadPayload reads until a newline or EOF, replaces invalid UTF-8, and parses.
// An empty request is an empty payload, not an error, so a bare connect-and-close
// is a no-op.
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

// replaceInvalidUTF8 is bytes.decode("utf-8", errors="replace"): Go strings hold
// arbitrary bytes, so invalid sequences would otherwise reach the JSON decoder.
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

// WriteResponse ignores a client that has already gone away: eww.yuck's handlers
// fire and forget, so a closed pipe is the normal end of a --quiet call.
func WriteResponse(conn io.Writer, reply Reply) {
	encoded, err := pyjson.Encode(reply, true)
	if err != nil {
		encoded = `{"ok":false,"error":"unencodable response"}`
	}
	_, _ = io.WriteString(conn, encoded+"\n")
}

func ServeConnection(store *state.Store, conn net.Conn) {
	defer conn.Close()

	reply := func() Reply {
		payload, err := ReadPayload(conn)
		if err != nil {
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

// persistWallpaperPick persists the reply's wallpaper.current rather than the
// requested path: current is what awww reports on screen, so a swallowed awww
// failure re-records the surviving wallpaper instead of one that never appeared.
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

// Listen prepares the control socket and returns a listener, split from Serve so
// a caller can bind without entering the accept loop.
func Listen() (net.Listener, error) {
	socketPath := paths.ControlSocket()

	// A stale socket from a killed daemon would make bind fail with EADDRINUSE. This
	// unlinks whatever is at that path, so never run it against the live runtime
	// directory of a daemon that is serving.
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
	// Owner only, best effort: a socket that cannot be chmodded is still
	// better than no control channel.
	_ = os.Chmod(socketPath, 0o600)
	return listener, nil
}

// Serve accepts forever and hands each connection to a bounded pool.
//
// accept() must never wait on a handler: eww.yuck binds eww-barctl to :onscroll,
// and a serve-then-accept loop lets the backlog fill until the kernel refuses
// connections with EAGAIN, which --quiet swallows entirely.
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
		// The goroutine waits for a slot; Accept does not. Reversing that
		// reintroduces the dropped-scroll bug.
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

// WriteBackendPidfile returns the cleanup the caller should run at shutdown. The
// cleanup re-reads the file and removes it only if it still holds OUR pid, so a
// daemon starting while an old one exits does not lose its pidfile to the corpse.
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
