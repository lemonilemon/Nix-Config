import json
import os
import subprocess
from pathlib import Path


EMPTY_MODULE = {"text": "", "tooltip": "", "class": ""}
BATTERY_DEFAULT = {"text": "󰂄", "alt": "󰂄", "capacity": 100, "class": "charging"}
MEDIA_DEFAULT = {"text": "", "status": "", "icon": "", "class": "", "tooltip": ""}
CCUSAGE_DEFAULT = {
    "text": "󱃖 --",
    "tooltip": "ccusage has no usage data yet",
    "class": "missing",
    "updated": "",
    "today": {
        "tokens": "--",
        "cost": "--",
        "input": "--",
        "output": "--",
        "cache": "--",
    },
    "block": {
        "tokens": "--",
        "projected": "--",
        "percent": 0,
        "remaining": "--",
        "cost": "--",
        "projected_cost": "--",
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


def backend_pidfile_path():
    return runtime_file("eww-backend.pid")


def control_socket_path():
    return runtime_file("eww-backend.sock")
