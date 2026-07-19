import os
import select
import socket
import subprocess
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


def watch_hyprland(state):
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
        except Exception:
            time.sleep(2)


def watch_command(state, key, collector, command):
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
