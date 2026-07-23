import subprocess
import threading
from pathlib import Path

from .common import parse_json, run_text, truncate_text

NOTIFICATIONS_DEFAULT = {"paused": "false", "new": 0, "count": 0, "groups": []}

APP_MAX = 20
SUMMARY_MAX = 48
BODY_MAX = 64

# Ephemeral UI state, process-lifetime only (same idea as the display-mode
# file, but collapse/badge state is fine to lose on daemon restart).
_UI_LOCK = threading.Lock()
_COLLAPSED = set()
_LAST_SEEN_US = 0


def _field(item, name, default=""):
    value = item.get(name)
    if isinstance(value, dict):
        return value.get("data", default)
    return default


def parse_history_items(history_json):
    # `dunstctl history` wraps everything in {"type": "aa{sv}", "data": [[...]]}
    # and every field in {"type": ..., "data": ...}. timestamp is MICROSECONDS
    # on the monotonic boot clock, not wall time.
    body = parse_json(history_json, {})
    if not isinstance(body, dict):
        return []
    data = body.get("data")
    if not (isinstance(data, list) and data and isinstance(data[0], list)):
        return []
    items = []
    for raw in data[0]:
        if not isinstance(raw, dict):
            continue
        try:
            item_id = int(_field(raw, "id", 0))
            timestamp = int(_field(raw, "timestamp", 0))
        except (TypeError, ValueError):
            continue
        items.append(
            {
                "id": item_id,
                "app": str(_field(raw, "appname") or "unknown"),
                "summary": str(_field(raw, "summary") or ""),
                "body": str(_field(raw, "body") or ""),
                "urgency": str(_field(raw, "urgency") or "NORMAL"),
                "timestamp": timestamp,
            }
        )
    items.sort(key=lambda item: -item["timestamp"])
    return items


def format_age(seconds):
    if seconds < 10:
        return "now"
    if seconds < 60:
        return f"{int(seconds)}s"
    if seconds < 3600:
        return f"{int(seconds // 60)}m"
    if seconds < 86400:
        return f"{int(seconds // 3600)}h"
    return f"{int(seconds // 86400)}d"


def uptime_seconds():
    try:
        return float(Path("/proc/uptime").read_text().split()[0])
    except Exception:
        return 0.0


def notifications_state_from_parts(items, paused_text, now_boot_us, collapsed, last_seen_us):
    grouped = {}
    order = []
    for item in items:
        app = truncate_text(item["app"], APP_MAX)
        if app not in grouped:
            grouped[app] = []
            order.append(app)
        age = max(0.0, (now_boot_us - item["timestamp"]) / 1_000_000)
        grouped[app].append(
            {
                "id": item["id"],
                "summary": truncate_text(item["summary"], SUMMARY_MAX),
                "body": truncate_text(item["body"].replace("\n", " "), BODY_MAX),
                "age": format_age(age),
                "urgency": item["urgency"],
            }
        )
    groups = [
        {
            "app": app,
            "count": len(grouped[app]),
            "collapsed": "true" if app in collapsed else "false",
            "items": grouped[app],
        }
        for app in order
    ]
    return {
        "paused": "true" if paused_text.strip() == "true" else "false",
        "new": sum(1 for item in items if item["timestamp"] > last_seen_us),
        "count": len(items),
        "groups": groups,
    }


def notifications_state():
    items = parse_history_items(run_text(["dunstctl", "history"]))
    paused_text = run_text(["dunstctl", "is-paused"])
    with _UI_LOCK:
        collapsed = set(_COLLAPSED)
        last_seen = _LAST_SEEN_US
    return notifications_state_from_parts(
        items, paused_text, uptime_seconds() * 1_000_000, collapsed, last_seen
    )
