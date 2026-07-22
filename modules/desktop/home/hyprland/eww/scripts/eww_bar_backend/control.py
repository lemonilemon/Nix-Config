import atexit
import json
import os
import socket
import subprocess
import sys
import threading

from .collectors import bluetooth_state, media_state, refresh_ai_usage, volume_state
from .common import backend_pidfile_path, control_socket_path, parse_json
from .display import display_state, set_display_mode
from .inhibitors import idle_inhibited_state, set_idle_inhibited, toggle_idle_inhibited


CONTROL_USAGE = (
    "usage: eww-barctl ping | volume up|down|set <0-100>|mute|sink <name> | "
    "media play-pause|next|previous | "
    "idle toggle|on|off|status | display normal|external|headless|restore|toggle|status | ai refresh"
    " | bluetooth power-toggle|disconnect <mac>"
)

_AI_REFRESH_LOCK = threading.Lock()


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
    return volume_state()


def set_volume(value):
    try:
        level = max(0, min(100, int(round(float(value)))))
    except (TypeError, ValueError):
        raise ValueError("volume set requires a number 0-100")
    subprocess.run(
        ["wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", f"{level}%"],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return volume_state()


def toggle_mute():
    subprocess.run(
        ["wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle"],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return volume_state()


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

    raise ValueError("unknown control command")


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
        server.listen(8)
        while True:
            conn, _addr = server.accept()
            with conn:
                try:
                    response = handle_control_command(state, read_control_payload(conn))
                except Exception as exc:
                    response = {"ok": False, "error": str(exc)}
                write_control_response(conn, response)


def control_payload_from_args(args):
    if not args or args[0] in ("-h", "--help", "help"):
        raise ValueError(CONTROL_USAGE)
    if args[0] == "ping":
        return {"command": "ping"}
    if args[0] == "volume" and len(args) >= 2:
        payload = {"command": "volume", "action": args[1]}
        if args[1] == "set" and len(args) >= 3:
            payload["value"] = args[2]
        elif args[1] == "sink" and len(args) >= 3:
            payload["sink"] = args[2]
        return payload
    if args[0] == "media" and len(args) == 2:
        return {"command": "media", "action": args[1]}
    if args[0] == "bluetooth" and len(args) >= 2:
        payload = {"command": "bluetooth", "action": args[1]}
        if args[1] == "disconnect" and len(args) >= 3:
            payload["mac"] = args[2]
        return payload
    if args[0] == "idle":
        action = args[1] if len(args) > 1 else "toggle"
        return {"command": "idle", "action": action}
    if args[0] == "display":
        action = args[1] if len(args) > 1 else "status"
        return {"command": "display", "action": action}
    if args[0] == "ai" and len(args) == 2:
        return {"command": "ai", "action": args[1]}
    raise ValueError(CONTROL_USAGE)


def send_control_command(payload):
    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
        client.connect(str(control_socket_path()))
        client.sendall((json.dumps(payload, separators=(",", ":")) + "\n").encode("utf-8"))
        client.shutdown(socket.SHUT_WR)
        chunks = []
        while True:
            chunk = client.recv(4096)
            if not chunk:
                break
            chunks.append(chunk)
    text = b"".join(chunks).decode("utf-8", errors="replace").strip()
    return parse_json(text, {"ok": False, "error": "invalid backend response"})


def run_ctl(args):
    quiet = False
    if args and args[0] in ("-q", "--quiet"):
        quiet = True
        args = args[1:]
    try:
        response = send_control_command(control_payload_from_args(args))
    except Exception as exc:
        response = {"ok": False, "error": str(exc)}
    if not quiet:
        print(json.dumps(response, separators=(",", ":")), flush=True)
    return 0 if response.get("ok") else 1
