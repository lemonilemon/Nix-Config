package watch

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"ewwbar/internal/collect"
	"ewwbar/internal/state"
)

// TestMain installs an empty collect fixture, the same rail package control
// uses. These tests drive watchers, and a watcher's gather function shells out.
func TestMain(m *testing.M) {
	restore := collect.InstallFixture(map[string]string{})
	code := m.Run()
	restore()
	os.Exit(code)
}

func TestHyprlandSocketPathNeedsASignature(t *testing.T) {
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
	if _, ok := HyprlandSocketPath(); ok {
		t.Fatal("no signature should mean no socket path")
	}

	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "sig123")
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	got, ok := HyprlandSocketPath()
	if !ok || got != "/run/user/1000/hypr/sig123/.socket2.sock" {
		t.Fatalf("socket path = %q, ok = %v", got, ok)
	}
}

func TestEmitLoopWritesOnEveryChange(t *testing.T) {
	store := state.New()
	reader, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go EmitLoop(ctx, store, writer)

	lines := make(chan string, 8)
	go func() {
		buf := make([]byte, 1<<16)
		var pending string
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				pending += string(buf[:n])
				for {
					index := strings.IndexByte(pending, '\n')
					if index < 0 {
						break
					}
					lines <- pending[:index]
					pending = pending[index+1:]
				}
			}
			if err != nil {
				return
			}
		}
	}()

	first := receive(t, lines)
	if !strings.Contains(first, `"submap":""`) {
		t.Fatalf("first emit was not the seed snapshot: %s", first[:80])
	}

	store.Update(func(bar *state.Bar) { bar.Submap = "resize" })
	second := receive(t, lines)
	if !strings.Contains(second, `"submap":"resize"`) {
		t.Fatalf("change was not emitted: %s", second[:80])
	}
}

func TestEmitLoopIgnoresAnUnchangedUpdate(t *testing.T) {
	// BarState.update compares before signalling, so writing the same value
	// must not wake the emit loop. Without that the pactl watcher alone would
	// rewrite the whole snapshot dozens of times a minute.
	store := state.New()
	store.Update(func(bar *state.Bar) { bar.Submap = "resize" })

	drained := false
	select {
	case <-store.Changed():
		drained = true
	case <-time.After(50 * time.Millisecond):
	}
	if !drained {
		t.Fatal("a real change should have signalled")
	}

	store.Update(func(bar *state.Bar) { bar.Submap = "resize" })
	select {
	case <-store.Changed():
		t.Fatal("an identical update signalled a change")
	case <-time.After(50 * time.Millisecond):
	}
}

// TestWatchCommandDebouncesABurst is the property the 200 ms window exists for:
// a subsystem that emits several events for one user action must cause ONE
// collection, not one per event.
func TestWatchCommandDebouncesABurst(t *testing.T) {
	previous := startCommand
	t.Cleanup(func() { startCommand = previous })

	startCommand = func(string, ...string) (io.ReadCloser, func(), error) {
		reader, writer := io.Pipe()
		go func() {
			for range 20 {
				_, _ = io.WriteString(writer, "event\n")
			}
			// Stay open: closing here would end the watcher before the
			// debounce window elapses and the test would measure nothing.
			time.Sleep(2 * time.Second)
			_ = writer.Close()
		}()
		return reader, func() { _ = reader.Close() }, nil
	}

	var mu sync.Mutex
	collections := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go WatchCommand(ctx, state.New(), nil, func() Update {
		mu.Lock()
		collections++
		mu.Unlock()
		return nil
	}, "fake")

	time.Sleep(900 * time.Millisecond)
	cancel()

	mu.Lock()
	got := collections
	mu.Unlock()

	// One for the seed, one for the whole burst. Anything approaching 20 means
	// the debounce is not collapsing the burst.
	if got > 3 {
		t.Fatalf("20 events caused %d collections; the burst was not debounced", got)
	}
	if got == 0 {
		t.Fatal("the burst caused no collection at all")
	}
}

func TestWatchCommandHonoursTheLineFilter(t *testing.T) {
	previous := startCommand
	t.Cleanup(func() { startCommand = previous })

	startCommand = func(string, ...string) (io.ReadCloser, func(), error) {
		reader, writer := io.Pipe()
		go func() {
			for range 5 {
				_, _ = io.WriteString(writer, "ignore me\n")
			}
			time.Sleep(2 * time.Second)
			_ = writer.Close()
		}()
		return reader, func() { _ = reader.Close() }, nil
	}

	var mu sync.Mutex
	collections := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go WatchCommand(ctx, state.New(), func(string) bool { return false }, func() Update {
		mu.Lock()
		collections++
		mu.Unlock()
		return nil
	}, "fake")

	time.Sleep(700 * time.Millisecond)
	cancel()

	mu.Lock()
	got := collections
	mu.Unlock()

	// Exactly the seed. A filter that rejects everything must leave the
	// watcher idle -- this is what stops the volume watcher feeding itself.
	if got != 1 {
		t.Fatalf("a reject-all filter still caused %d collections (want 1, the seed)", got)
	}
}

func receive(t *testing.T, lines <-chan string) string {
	t.Helper()
	select {
	case line := <-lines:
		return line
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for an emit")
		return ""
	}
}
