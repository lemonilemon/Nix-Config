"""Control client for eww-barctl.

Deliberately isolated from the daemon's import graph. scripts/backend dispatches
here before importing anything else, so a bar click costs one socket round-trip
instead of loading collectors -> urllib.request -> http.client -> email.parser.
The test in tests/eww_bar_backend/test_ctl.py enforces that importing this
module never pulls in subprocess, common, collectors, or app.
"""

import json
import socket

from .paths import control_socket_path


CONTROL_USAGE = (
    "usage: eww-barctl ping | volume up|down|set <0-100>|mute|sink <name> | "
    "media play-pause|next|previous | "
    "idle toggle|on|off|status | display normal|external|headless|restore|toggle|status | ai refresh"
    " | bluetooth power-toggle|disconnect <mac> | network wifi-toggle"
    " | notif toggle-group <app>|dismiss <id>|clear-group <app>|clear-all|dnd-toggle|mark-seen"
    " | wallpaper set <path>|rescan"
)


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
    if args[0] == "network" and len(args) >= 2:
        return {"command": "network", "action": args[1]}
    if args[0] == "idle":
        action = args[1] if len(args) > 1 else "toggle"
        return {"command": "idle", "action": action}
    if args[0] == "display":
        action = args[1] if len(args) > 1 else "status"
        return {"command": "display", "action": action}
    if args[0] == "ai" and len(args) == 2:
        return {"command": "ai", "action": args[1]}
    if args[0] == "notif" and len(args) >= 2:
        payload = {"command": "notif", "action": args[1]}
        if args[1] in ("toggle-group", "clear-group") and len(args) >= 3:
            payload["app"] = " ".join(args[2:])
        elif args[1] == "dismiss" and len(args) >= 3:
            payload["id"] = args[2]
        return payload
    if args[0] == "wallpaper" and len(args) >= 2:
        payload = {"command": "wallpaper", "action": args[1]}
        if args[1] == "set" and len(args) >= 3:
            payload["path"] = args[2]
        return payload
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
    # Mirrors common.parse_json. Inlined rather than imported: it only needs
    # json, and a divergence here would degrade an error message rather than
    # misroute a socket (unlike control_socket_path, which is shared via paths.py).
    try:
        return json.loads(text)
    except Exception:
        return {"ok": False, "error": "invalid backend response"}


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
