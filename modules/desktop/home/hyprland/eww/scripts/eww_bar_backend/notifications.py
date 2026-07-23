import subprocess
import threading
import time

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


def boottime_seconds():
    # dunst stamps history with CLOCK_BOOTTIME (suspend-included) — verified on
    # this host: timestamps run ahead of CLOCK_MONOTONIC by exactly the
    # accumulated suspend time. Using MONOTONIC here makes ages go negative
    # (clamped to "now") after any suspend.
    try:
        return time.clock_gettime(time.CLOCK_BOOTTIME)
    except Exception:
        try:
            return time.clock_gettime(time.CLOCK_MONOTONIC)
        except Exception:
            return 0.0


def notifications_state_from_parts(items, paused_text, now_monotonic_us, collapsed, last_seen_us):
    grouped = {}
    order = []
    for item in items:
        app = truncate_text(item["app"], APP_MAX)
        if app not in grouped:
            grouped[app] = []
            order.append(app)
        age = max(0.0, (now_monotonic_us - item["timestamp"]) / 1_000_000)
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
        items, paused_text, boottime_seconds() * 1_000_000, collapsed, last_seen
    )


def _run_dunstctl(args):
    subprocess.run(
        ["dunstctl", *args],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        check=False, timeout=5,
    )


def toggle_group(app):
    if not app:
        raise ValueError("notif toggle-group requires an app name")
    with _UI_LOCK:
        if app in _COLLAPSED:
            _COLLAPSED.discard(app)
        else:
            _COLLAPSED.add(app)
    return notifications_state()


def dismiss_notification(notif_id):
    try:
        value = int(notif_id)
    except (TypeError, ValueError):
        raise ValueError("notif dismiss requires a numeric id")
    _run_dunstctl(["history-rm", str(value)])
    return notifications_state()


def clear_group(app):
    if not app:
        raise ValueError("notif clear-group requires an app name")
    # Group names are the truncated app names, so match with the same truncation.
    for item in parse_history_items(run_text(["dunstctl", "history"])):
        if truncate_text(item["app"], APP_MAX) == app:
            _run_dunstctl(["history-rm", str(item["id"])])
    return notifications_state()


def clear_all_notifications():
    _run_dunstctl(["history-clear"])
    return notifications_state()


def toggle_dnd():
    _run_dunstctl(["set-paused", "toggle"])
    return notifications_state()


def mark_seen():
    global _LAST_SEEN_US
    items = parse_history_items(run_text(["dunstctl", "history"]))
    newest = items[0]["timestamp"] if items else int(boottime_seconds() * 1_000_000)
    with _UI_LOCK:
        _LAST_SEEN_US = max(_LAST_SEEN_US, newest)
    return notifications_state()
