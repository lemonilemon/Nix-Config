import json
import os
import subprocess
from pathlib import Path


EMPTY_MODULE = {"text": "", "tooltip": "", "class": ""}
NETWORK_DEFAULT = {"text": "", "tooltip": "", "class": "", "wifi_enabled": "false"}
BATTERY_DEFAULT = {
    "text": "󰂄",
    "alt": "󰂄",
    "capacity": 100,
    "class": "charging",
    "status": "Unknown",
    "time": "N/A",
    "health": "--",
    "power": "—",
}
MEDIA_DEFAULT = {"text": "", "status": "", "icon": "", "class": "", "tooltip": ""}
VOLUME_DEFAULT = {"text": "", "percent": 0, "muted": "false", "class": "", "sinks": []}
ACTIVE_WINDOW_DEFAULT = {"text": "", "tooltip": "", "class": ""}
BLUETOOTH_DEFAULT = {"text": "", "tooltip": "", "class": "", "powered": "false", "devices": []}
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


def module_enabled(name):
    """Whether a bar module is switched on.

    default.nix writes exactly "1" or "0" into EWW_BAR_<NAME> on the eww-bar
    service, so only that literal "0" disables. An absent variable means
    enabled, which keeps the backend usable when run by hand outside the unit.
    """
    return os.environ.get(f"EWW_BAR_{name}", "1") != "0"


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
