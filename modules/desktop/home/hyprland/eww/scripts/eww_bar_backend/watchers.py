import os
import select
import socket
import subprocess
import threading
import time
from pathlib import Path

from .collectors import (
    active_window_state,
    bluetooth_state,
    network_state,
    submap_from_event,
    volume_state,
    workspace_state,
)
from .common import parse_json, run_text
from .display import display_state
from .inhibitors import idle_inhibited_state


def periodic(state, interval, **collectors):
    while True:
        values = {}
        for key, collector in collectors.items():
            values[key] = collector()
        state.update(**values)
        time.sleep(interval)


def periodic_refresh(state, interval, refresh):
    # Like `periodic`, but the collector owns emitting to `state` (so it can flag
    # progress and update incrementally). Runs once immediately, then every interval.
    while True:
        try:
            refresh(state)
        except Exception:
            pass
        time.sleep(interval)


def watch_idle_inhibitor(state, refresh_event):
    state.update(idle_inhibited=idle_inhibited_state(), display=display_state())
    while True:
        refresh_event.wait(timeout=30)
        refresh_event.clear()
        state.update(idle_inhibited=idle_inhibited_state(), display=display_state())


def hyprland_socket_path():
    runtime_dir = os.environ.get("XDG_RUNTIME_DIR", f"/run/user/{os.getuid()}")
    signature = os.environ.get("HYPRLAND_INSTANCE_SIGNATURE", "")
    if not signature:
        return None
    return Path(runtime_dir) / "hypr" / signature / ".socket2.sock"


def monitor_event(line):
    # Hyprland emits both v1 and v2 monitor events; matching only the v1
    # prefixes (v2 lines start "monitoraddedv2>>") keeps this single-fire.
    for prefix, action in (("monitoradded>>", "added"), ("monitorremoved>>", "removed")):
        if line.startswith(prefix):
            name = line[len(prefix):].strip()
            if name:
                return (action, name)
    return None


def bar_window_command(action, name):
    # Must mirror the open-bars script in eww/default.nix so hotplugged
    # monitors get the same bar-<name> windows as service startup.
    if action == "added":
        return ["eww", "open", "bar", "--id", f"bar-{name}", "--screen", name, "--arg", f"output={name}"]
    return ["eww", "close", f"bar-{name}"]


def apply_monitor_event(action, name):
    # GDK learns about a hotplugged output a beat after Hyprland announces it,
    # so retry the open a few times instead of trusting the first attempt.
    attempts = 5 if action == "added" else 1
    for _ in range(attempts):
        time.sleep(1)
        # Another opener may have won meanwhile (the startup script, or eww's
        # own config reload) — re-opening an open id makes the bar flicker.
        if action == "added" and name in open_bar_names(run_text(["eww", "active-windows"])):
            return
        try:
            result = subprocess.run(
                bar_window_command(action, name),
                stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                check=False, timeout=10,
            )
            if result.returncode == 0:
                return
        except Exception:
            pass


def open_bar_names(active_windows_text):
    names = set()
    for line in active_windows_text.splitlines():
        window_id = line.split(":", 1)[0].strip()
        if window_id.startswith("bar-"):
            names.add(window_id[len("bar-"):])
    return names


def missing_bar_monitors(monitors_json_text, active_windows_text):
    monitors = parse_json(monitors_json_text, [])
    if not isinstance(monitors, list):
        return []
    names = [m.get("name") for m in monitors if isinstance(m, dict) and m.get("name")]
    open_names = open_bar_names(active_windows_text)
    return [name for name in names if name not in open_names]


def reconcile_bar_windows():
    # When a monitor connects, eww reloads its whole configuration, which
    # kills and respawns this backend — the monitoradded event fires exactly
    # while no listener is alive, so it can never be caught. Instead, on
    # every backend start compare live monitors against open bar windows and
    # open whatever is missing.
    for name in missing_bar_monitors(
        run_text(["hyprctl", "monitors", "-j"], timeout=5.0),
        run_text(["eww", "active-windows"], timeout=5.0),
    ):
        apply_monitor_event("added", name)


def watch_hyprland(state):
    threading.Thread(target=reconcile_bar_windows, daemon=True).start()
    while True:
        state.update(workspace_state=workspace_state())
        socket_path = hyprland_socket_path()
        if socket_path is None or not socket_path.exists():
            time.sleep(2)
            continue

        try:
            with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as sock:
                sock.connect(str(socket_path))
                file_obj = sock.makefile("r", encoding="utf-8", errors="replace")
                for line in file_obj:
                    line = line.rstrip("\n")
                    submap = submap_from_event(line)
                    if submap is not None:
                        state.update(submap=submap)
                    if line.startswith(
                        (
                            "workspace>>",
                            "focusedmon>>",
                            "openwindow>>",
                            "closewindow>>",
                            "movewindow>>",
                            "createworkspace>>",
                            "destroyworkspace>>",
                            "urgent>>",
                        )
                    ):
                        state.update(workspace_state=workspace_state())
                    if line.startswith(
                        (
                            "activewindow>>",
                            "activewindowv2>>",
                            "closewindow>>",
                        )
                    ):
                        state.update(active_window=active_window_state())
                    event = monitor_event(line)
                    if event is not None:
                        # Own thread: apply_monitor_event sleeps/retries and
                        # must not stall workspace/window event handling.
                        threading.Thread(
                            target=apply_monitor_event, args=event, daemon=True
                        ).start()
        except Exception:
            time.sleep(2)


def watch_command(state, key, collector, command, line_filter=None):
    """Re-collect `key` whenever `command` prints a line.

    `line_filter` drops lines that cannot have changed anything the bar renders.
    Without one, a watcher whose collector shells out to the same subsystem it
    is watching becomes its own event source -- see volume_event_is_relevant.
    """
    state.update(**{key: collector()})
    while True:
        proc = None
        try:
            proc = subprocess.Popen(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,
                text=True,
            )
            assert proc.stdout is not None
            for _line in proc.stdout:
                if line_filter is not None and not line_filter(_line):
                    continue
                time.sleep(0.2)
                while True:
                    readable, _, _ = select.select([proc.stdout], [], [], 0.2)
                    if not readable:
                        break
                    proc.stdout.readline()
                state.update(**{key: collector()})
            proc.wait(timeout=1)
        except Exception:
            if proc is not None:
                try:
                    proc.kill()
                except Exception:
                    pass
        time.sleep(2)


def watch_bluetooth(state):
    state.update(bluetooth=bluetooth_state())
    while True:
        proc = None
        try:
            proc = subprocess.Popen(
                ["bluetoothctl", "--monitor"],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,
                text=True,
            )
            assert proc.stdout is not None
            for _line in proc.stdout:
                time.sleep(0.2)
                while True:
                    readable, _, _ = select.select([proc.stdout], [], [], 0.2)
                    if not readable:
                        break
                    proc.stdout.readline()
                state.update(bluetooth=bluetooth_state())
        except Exception:
            pass
        finally:
            if proc is not None:
                try:
                    proc.kill()
                except Exception:
                    pass
        time.sleep(2)


def emit_loop(state):
    try:
        print(state.snapshot(), flush=True)
    except BrokenPipeError:
        return
    while True:
        state.changed.wait()
        state.changed.clear()
        try:
            print(state.snapshot(), flush=True)
        except BrokenPipeError:
            return
