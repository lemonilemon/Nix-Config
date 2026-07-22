import json
import os
import subprocess
from pathlib import Path


EMPTY_MODULE = {"text": "", "tooltip": "", "class": ""}
BATTERY_DEFAULT = {"text": "󰂄", "alt": "󰂄", "capacity": 100, "class": "charging"}
MEDIA_DEFAULT = {"text": "", "status": "", "icon": "", "class": "", "tooltip": ""}
VOLUME_DEFAULT = {"text": "", "percent": 0, "muted": "false", "class": "", "sinks": []}
ACTIVE_WINDOW_DEFAULT = {"text": "", "tooltip": "", "class": ""}
AI_USAGE_DEFAULT = {
    "text": "󰙴 --",
    "tooltip": "AI usage data is not available yet",
    "class": "missing",
    "source": "missing",
    "updated": "",
    "periods": {
        "today": {"label": "Today", "range": "", "tokens": "--", "cost": "--", "agents": []},
        "week": {"label": "This week", "range": "", "tokens": "--", "cost": "--", "agents": []},
        "month": {"label": "This month", "range": "", "tokens": "--", "cost": "--", "agents": []},
    },
    "agents": "--",
    "quotas": [
        {
            "key": "claude",
            "name": "Claude",
            "plan": "--",
            "status": "waiting",
            "class": "missing",
            "updated": "",
            "windows": [],
            "meta": [],
        },
        {
            "key": "codex",
            "name": "Codex",
            "plan": "--",
            "status": "waiting",
            "class": "missing",
            "updated": "",
            "windows": [],
            "meta": [],
        },
        {
            "key": "antigravity",
            "name": "Antigravity",
            "plan": "--",
            "status": "waiting",
            "class": "missing",
            "updated": "",
            "windows": [],
            "meta": [],
        },
    ],
    "meta": {
        "pricing": "offline",
        "refresh": "5m",
        "status": "waiting",
        "stale": "false",
        "refreshing": "false",
    },
}
WORKSPACE_DEFAULT = {
    "ws1_class": "empty",
    "ws1_text": "",
    "ws2_class": "empty",
    "ws2_text": "",
    "ws3_class": "empty",
    "ws3_text": "",
    "ws4_class": "empty",
    "ws4_text": "",
    "ws5_class": "empty",
    "ws5_text": "",
}


def run_text(command, timeout=2.0):
    try:
        return subprocess.check_output(
            command,
            text=True,
            stderr=subprocess.DEVNULL,
            timeout=timeout,
        )
    except Exception:
        return ""


def parse_json(text, default):
    try:
        return json.loads(text)
    except Exception:
        return default


def truncate_text(text, max_len):
    if len(text) <= max_len:
        return text
    return text[: max_len - 3] + "..."


def runtime_file(name):
    runtime_dir = os.environ.get("XDG_RUNTIME_DIR", "/tmp")
    return Path(runtime_dir) / name


def idle_pidfile_path():
    return runtime_file("eww-idle-inhibit.pid")


def display_mode_path():
    return runtime_file("eww-display-mode")


def backend_pidfile_path():
    return runtime_file("eww-backend.pid")


def control_socket_path():
    return runtime_file("eww-backend.sock")
