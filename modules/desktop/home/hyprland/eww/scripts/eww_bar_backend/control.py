import atexit
import concurrent.futures
import json
import os
import socket
import subprocess
import sys
import threading

from .collectors import (
    bluetooth_state,
    media_state,
    network_radio_enabled,
    network_state,
    refresh_ai_usage,
    volume_state,
)
from .common import backend_pidfile_path, control_socket_path
from .display import display_state, set_display_mode
from .inhibitors import idle_inhibited_state, set_idle_inhibited, toggle_idle_inhibited
from .notifications import (
    clear_all_notifications,
    clear_group,
    dismiss_notification,
    mark_seen,
    toggle_dnd,
    toggle_group,
)
from .wallpaper import set_wallpaper, wallpaper_state


_AI_REFRESH_LOCK = threading.Lock()

# Enough to absorb a scroll gesture without letting a stuck handler (every one
# of them shells out) spawn threads without bound. Excess connections queue in
# the pool rather than being refused by the kernel.
CONTROL_WORKERS = 8


def write_backend_pidfile():
    pidfile = backend_pidfile_path()
    pid = str(os.getpid())
    try:
        pidfile.parent.mkdir(parents=True, exist_ok=True)
        pidfile.write_text(pid)
    except Exception:
        return

    def cleanup():
        try:
            if pidfile.read_text().strip() == pid:
                pidfile.unlink()
        except Exception:
            pass

    atexit.register(cleanup)


def adjust_volume(direction):
    if direction == "up":
        amount = "2%+"
    elif direction == "down":
        amount = "2%-"
    else:
        raise ValueError("volume direction must be up or down")
    subprocess.run(
        ["wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", amount],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    # refresh_sinks=False: a level nudge cannot add, remove or re-default a
    # sink, and this is the :onscroll path where the two extra pactl forks are
    # the whole cost.
    return volume_state(refresh_sinks=False)


def set_volume(value):
    try:
        level = max(0, min(100, int(round(float(value)))))
    except (TypeError, ValueError):
        raise ValueError("volume set requires a number 0-100")
    subprocess.run(
        ["wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", f"{level}%"],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return volume_state(refresh_sinks=False)


def toggle_mute():
    subprocess.run(
        ["wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle"],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return volume_state(refresh_sinks=False)


def set_sink(name):
    if not name:
        raise ValueError("volume sink requires a sink name")
    subprocess.run(
        ["pactl", "set-default-sink", name],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return volume_state()


def control_media(action):
    allowed_actions = {"play-pause", "next", "previous"}
    if action not in allowed_actions:
        raise ValueError("media action must be play-pause, next, or previous")
    subprocess.run(
        ["playerctl", action],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    return media_state()


def toggle_bluetooth_power():
    current = bluetooth_state().get("powered", "false")
    target = "off" if current == "true" else "on"
    subprocess.run(
        ["bluetoothctl", "power", target],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return bluetooth_state()


def disconnect_bluetooth(mac):
    if not mac:
        raise ValueError("bluetooth disconnect requires a device address")
    subprocess.run(
        ["bluetoothctl", "disconnect", mac],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return bluetooth_state()


def toggle_wifi():
    target = "off" if network_radio_enabled() == "true" else "on"
    subprocess.run(
        ["nmcli", "radio", "wifi", target],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return network_state()


def queue_ai_refresh(state):
    if not _AI_REFRESH_LOCK.acquire(blocking=False):
        return False

    def refresh():
        try:
            refresh_ai_usage(state)
        finally:
            _AI_REFRESH_LOCK.release()

    threading.Thread(target=refresh, daemon=True).start()
    return True


def handle_control_command(state, payload):
    command = payload.get("command")
    if command == "ping":
        return {"ok": True, "command": "ping"}

    if command == "volume":
        action = payload.get("action", "")
        if action in ("up", "down"):
            value = adjust_volume(action)
        elif action == "set":
            value = set_volume(payload.get("value"))
        elif action == "mute":
            value = toggle_mute()
        elif action == "sink":
            value = set_sink(payload.get("sink", ""))
        else:
            raise ValueError("volume action must be up, down, set, mute, or sink")
        state.update(volume=value)
        return {"ok": True, "command": "volume", "action": action, "volume": value}

    if command == "media":
        action = payload.get("action", "")
        value = control_media(action)
        state.update(media=value)
        return {"ok": True, "command": "media", "action": action, "media": value}

    if command == "bluetooth":
        action = payload.get("action", "")
        if action == "power-toggle":
            value = toggle_bluetooth_power()
        elif action == "disconnect":
            value = disconnect_bluetooth(payload.get("mac", ""))
        else:
            raise ValueError("bluetooth action must be power-toggle or disconnect")
        state.update(bluetooth=value)
        return {"ok": True, "command": "bluetooth", "action": action, "bluetooth": value}

    if command == "network":
        action = payload.get("action", "")
        if action == "wifi-toggle":
            value = toggle_wifi()
        else:
            raise ValueError("network action must be wifi-toggle")
        state.update(network=value)
        return {"ok": True, "command": "network", "action": action, "network": value}

    if command == "idle":
        action = payload.get("action", "toggle")
        if action == "toggle":
            value = toggle_idle_inhibited()
        elif action == "on":
            value = set_idle_inhibited(True)
        elif action == "off":
            value = set_idle_inhibited(False)
        elif action == "status":
            value = idle_inhibited_state()
        else:
            raise ValueError("idle action must be toggle, on, off, or status")
        state.update(idle_inhibited=value, display=display_state())
        return {"ok": True, "command": "idle", "idle_inhibited": value}

    if command == "display":
        action = payload.get("action", "status")
        value = set_display_mode(action)
        state.update(idle_inhibited=idle_inhibited_state(), display=value)
        return {"ok": True, "command": "display", "action": action, "display": value}

    if command == "ai":
        action = payload.get("action", "refresh")
        if action != "refresh":
            raise ValueError("ai action must be refresh")
        status = "queued" if queue_ai_refresh(state) else "already-refreshing"
        return {"ok": True, "command": "ai", "action": action, "status": status}

    if command == "notif":
        action = payload.get("action", "")
        if action == "toggle-group":
            value = toggle_group(payload.get("app", ""))
        elif action == "dismiss":
            value = dismiss_notification(payload.get("id", ""))
        elif action == "clear-group":
            value = clear_group(payload.get("app", ""))
        elif action == "clear-all":
            value = clear_all_notifications()
        elif action == "dnd-toggle":
            value = toggle_dnd()
        elif action == "mark-seen":
            value = mark_seen()
        else:
            raise ValueError(
                "notif action must be toggle-group, dismiss, clear-group, clear-all, dnd-toggle, or mark-seen"
            )
        state.update(notifications=value)
        return {"ok": True, "command": "notif", "action": action, "notifications": value}

    if command == "wallpaper":
        action = payload.get("action", "")
        if action == "set":
            value = set_wallpaper(payload.get("path", ""))
        elif action == "rescan":
            value = wallpaper_state()
        else:
            raise ValueError("wallpaper action must be set or rescan")
        state.update(wallpaper=value)
        return {"ok": True, "command": "wallpaper", "action": action, "wallpaper": value}

    raise ValueError("unknown control command")


def serve_control_connection(state, conn):
    with conn:
        try:
            response = handle_control_command(state, read_control_payload(conn))
        except Exception as exc:
            response = {"ok": False, "error": str(exc)}
        write_control_response(conn, response)


def read_control_payload(conn):
    chunks = []
    while True:
        chunk = conn.recv(4096)
        if not chunk:
            break
        chunks.append(chunk)
        if b"\n" in chunk:
            break
    text = b"".join(chunks).decode("utf-8", errors="replace").strip()
    if not text:
        return {}
    return json.loads(text)


def write_control_response(conn, payload):
    try:
        conn.sendall((json.dumps(payload, separators=(",", ":")) + "\n").encode("utf-8"))
    except (BrokenPipeError, ConnectionResetError):
        pass


def control_server(state):
    socket_path = control_socket_path()
    try:
        socket_path.unlink()
    except FileNotFoundError:
        pass
    except Exception as exc:
        print(f"eww-bar control server: unable to unlink {socket_path}: {exc}", file=sys.stderr, flush=True)
        return

    socket_path.parent.mkdir(parents=True, exist_ok=True)

    def cleanup():
        try:
            socket_path.unlink()
        except Exception:
            pass

    atexit.register(cleanup)

    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as server:
        try:
            server.bind(str(socket_path))
        except OSError as exc:
            print(f"eww-bar control server: unable to bind {socket_path}: {exc}", file=sys.stderr, flush=True)
            return
        try:
            os.chmod(socket_path, 0o600)
        except Exception:
            pass
        # Deep backlog and a worker pool, because accept() must never wait on a
        # handler. eww.yuck binds eww-barctl to :onscroll, and a scroll wheel
        # delivers ticks far faster than a command that shells out can be
        # served; the old serve-then-accept loop let the backlog fill and the
        # kernel then refused the rest with EAGAIN, which the client reports as a
        # failed command and --quiet swallows entirely. Measured against a
        # 40-client burst before this change: 30 dropped.
        server.listen(128)
        with concurrent.futures.ThreadPoolExecutor(
            max_workers=CONTROL_WORKERS, thread_name_prefix="eww-control"
        ) as pool:
            while True:
                try:
                    conn, _addr = server.accept()
                except OSError as exc:
                    print(f"eww-bar control server: accept failed: {exc}", file=sys.stderr, flush=True)
                    continue
                pool.submit(serve_control_connection, state, conn)
