import json
import os
import re
import time
import urllib.error
import urllib.request
from copy import deepcopy
from datetime import datetime
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path

from .common import (
    ACTIVE_WINDOW_DEFAULT,
    AI_USAGE_DEFAULT,
    BATTERY_DEFAULT,
    MEDIA_DEFAULT,
    VOLUME_DEFAULT,
    WORKSPACE_DEFAULT,
    idle_pidfile_path,
    parse_json,
    run_text,
    truncate_text,
)


def clock_state():
    now = time.localtime()
    return {
        "time": " " + time.strftime("%H:%M", now),
        "date": " " + time.strftime("%F", now),
        "tooltip": " " + time.strftime("%A, %F %Z", now),
    }


def workspace_state_from_json(active_json, workspaces_json, clients_json):
    active = parse_json(active_json, {})
    workspaces = parse_json(workspaces_json, [])
    clients = parse_json(clients_json, [])
    active_id = active.get("id", 0)

    state = {}
    for workspace_id in range(1, 6):
        occupied = any(
            item.get("id") == workspace_id and item.get("windows", 0) > 0
            for item in workspaces
            if isinstance(item, dict)
        )
        urgent = any(
            item.get("urgent") is True
            and isinstance(item.get("workspace"), dict)
            and item["workspace"].get("id") == workspace_id
            for item in clients
            if isinstance(item, dict)
        )

        if urgent and active_id == workspace_id:
            cls = "active urgent"
            text = ""
        elif urgent:
            cls = "occupied urgent"
            text = ""
        elif active_id == workspace_id:
            cls = "active"
            text = ""
        elif occupied:
            cls = "occupied"
            text = ""
        else:
            cls = "empty"
            text = ""

        state[f"ws{workspace_id}_class"] = cls
        state[f"ws{workspace_id}_text"] = text

    return state


def workspace_state():
    return workspace_state_from_json(
        run_text(["hyprctl", "activeworkspace", "-j"]),
        run_text(["hyprctl", "workspaces", "-j"]),
        run_text(["hyprctl", "clients", "-j"]),
    )


def submap_from_event(line):
    if not line.startswith("submap>>"):
        return None
    value = line.split(">>", 1)[1]
    if value in ("default", "reset"):
        return ""
    return value


APP_ICONS = {
    "kitty": " ",
    "zen-beta": "󰈹 ",
    "zen": "󰈹 ",
    "firefox": " ",
    "webcord": "󰙯 ",
    "spotify": " ",
    "org.remmina.Remmina": "󰢹 ",
    "thunar": " ",
    "code": "󰨞 ",
    "code-url-handler": "󰨞 ",
}


def active_window_state():
    data = parse_json(run_text(["hyprctl", "activewindow", "-j"]), {})
    cls = data.get("class", "")
    title = data.get("title", "")
    if not cls and not title:
        return ACTIVE_WINDOW_DEFAULT.copy()
    icon = APP_ICONS.get(cls, " ")
    # Upper bound only — the bar label has :truncate, so GTK ellipsizes to the
    # actual space available; this just keeps the state JSON bounded.
    display = f"{icon}{cls} · {truncate_text(title, 60)}" if title else f"{icon}{cls}"
    return {
        "text": display,
        "tooltip": f"{cls}\n{title}",
        "class": cls,
    }


def media_state_from_text(status_text, metadata_text):
    metadata = metadata_text.splitlines()
    if not metadata or not metadata[0].strip():
        return MEDIA_DEFAULT.copy()

    status_lines = status_text.splitlines()
    status = status_lines[0].strip() if status_lines else ""
    normalized = status.lower()
    if normalized == "playing":
        icon = ""
        cls = "playing"
        display_status = "Playing"
    elif normalized == "paused":
        icon = ""
        cls = "paused"
        display_status = "Paused"
    elif normalized == "stopped":
        icon = ""
        cls = "stopped"
        display_status = "Stopped"
    else:
        icon = ""
        cls = ""
        display_status = status or "Media"

    return {
        "text": truncate_text(metadata[0].strip(), 60),
        "status": display_status,
        "icon": icon,
        "class": cls,
        "tooltip": f"{display_status}\nLeft click: play/pause\nRight click: next",
    }


def media_state():
    return media_state_from_text(
        run_text(["playerctl", "status"], timeout=1.0),
        run_text(["playerctl", "metadata", "--format", "{{artist}} - {{title}}"]),
    )


def number_value(data, *keys):
    for key in keys:
        value = data.get(key)
        if isinstance(value, bool):
            continue
        if isinstance(value, (int, float)):
            return value
        if isinstance(value, str):
            try:
                return float(value)
            except ValueError:
                continue
    return 0


def list_value(data, *keys):
    for key in keys:
        value = data.get(key)
        if isinstance(value, list):
            return value
    return []


def format_tokens(value):
    if value <= 0:
        return "--"
    if value >= 1_000_000:
        return f"{value / 1_000_000:.1f}M"
    if value >= 1_000:
        return f"{value / 1_000:.1f}K"
    return f"{int(value)}"


def format_cost(value):
    if value <= 0:
        return "--"
    return f"${value:.2f}"


def format_clock_time(epoch):
    if epoch is None:
        return "--"
    return time.strftime("%F %H:%M", time.localtime(epoch))


def format_remaining(seconds):
    if seconds <= 0:
        return "0m"
    minutes = int(seconds // 60)
    hours, mins = divmod(minutes, 60)
    if hours:
        return f"{hours}h {mins}m"
    return f"{mins}m"


def percent_part(value, total):
    if value <= 0 or total <= 0:
        return 0
    return max(0, min(100, int(round(value * 100 / total))))


def clamp_percent(value):
    return max(0, min(100, int(round(value))))


def parse_iso_epoch(value):
    if not isinstance(value, str) or not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp()
    except Exception:
        pass
    try:
        return time.mktime(time.strptime(value[:19], "%Y-%m-%dT%H:%M:%S"))
    except Exception:
        return None


def daily_row_from_report(daily_report, now_epoch=None):
    today_date = time.strftime("%F", time.localtime(now_epoch or time.time()))
    daily_rows = list_value(daily_report, "daily", "data", "days", "rows")
    today_row = next(
        (
            row
            for row in daily_rows
            if isinstance(row, dict) and (row.get("date") or row.get("period")) == today_date
        ),
        None,
    )
    if today_row is None and daily_rows:
        today_row = next((row for row in reversed(daily_rows) if isinstance(row, dict)), None)
    return today_row or {}


def daily_token_values(today_row):
    input_tokens = number_value(today_row, "inputTokens", "input_tokens", "input")
    output_tokens = number_value(today_row, "outputTokens", "output_tokens", "output")
    cache_tokens = number_value(
        today_row,
        "cacheCreationTokens",
        "cacheCreationInputTokens",
        "cache_creation_input_tokens",
        "cache_creation_tokens",
    ) + number_value(
        today_row,
        "cacheReadTokens",
        "cacheReadInputTokens",
        "cache_read_input_tokens",
        "cache_read_tokens",
    )
    total_tokens = number_value(today_row, "totalTokens", "total_tokens", "tokens")
    if total_tokens <= 0:
        total_tokens = input_tokens + output_tokens + cache_tokens
    total_cost = number_value(today_row, "totalCost", "total_cost", "costUSD", "cost")
    return {
        "input": input_tokens,
        "output": output_tokens,
        "cache": cache_tokens,
        "total": total_tokens,
        "cost": total_cost,
    }


def agents_from_daily_json(daily_json, now_epoch=None):
    daily_report = parse_json(daily_json, {})
    today_row = daily_row_from_report(daily_report, now_epoch=now_epoch)
    if not today_row:
        return []

    agents = []
    metadata = today_row.get("metadata")
    if isinstance(metadata, dict):
        agents.extend(value for value in metadata.get("agents", []) if isinstance(value, str))
    agent = today_row.get("agent")
    if isinstance(agent, str) and agent != "all":
        agents.append(agent)
    return [agent.lower() for agent in agents]


def agents_text(agents):
    known = ("claude", "codex", "gemini")
    seen = [name for name in known if any(name in token for token in agents)]
    return " · ".join(seen) if seen else "--"


def quota_window_class(percent, has_reset):
    if percent >= 90:
        return "critical"
    if percent >= 80:
        return "warning"
    if percent > 0:
        return "active"
    if has_reset:
        return "empty"
    return "missing"


def quota_card_class(quota):
    classes = [window.get("class") for window in quota.get("windows", [])]
    classes.extend(item.get("class") for item in quota.get("meta", []))
    for name in ("critical", "warning", "active", "empty"):
        if name in classes:
            return name
    return "missing"


def quota_default(key, name, status="waiting"):
    for card in AI_USAGE_DEFAULT["quotas"]:
        if card["key"] == key:
            card = deepcopy(card)
            card["status"] = status
            return card
    return {
        "key": key,
        "name": name,
        "plan": "--",
        "status": status,
        "class": "missing",
        "updated": "",
        "windows": [],
        "meta": [],
    }


# Codex and Antigravity quotas come from openusage-cli (OpenUsage Community).
# Its plugins read the same CLI logins but ship their own client creds, so no
# OAuth plumbing (or ai-usage.env) is needed here. Claude stays on the native
# collector below: openusage's Claude plugin refreshes and rewrites the Claude
# Code token, which the read-only rule there deliberately avoids.
OPENUSAGE_PROVIDERS = (
    ("codex", "Codex"),
    ("antigravity", "Antigravity"),
)


def openusage_window(line, now_epoch=None):
    now = now_epoch or time.time()
    percent = percent_part(number_value(line, "used"), number_value(line, "limit"))
    reset_epoch = parse_iso_epoch(line.get("resetsAt"))
    has_reset = reset_epoch is not None and reset_epoch > 0
    return {
        "label": truncate_text(str(line.get("label") or "?"), 22),
        "percent": percent,
        "value": f"{percent}%",
        "remaining": format_remaining(max(0, reset_epoch - now)) if has_reset and reset_epoch > now else "--",
        "reset": format_clock_time(reset_epoch) if has_reset else "--",
        "class": quota_window_class(percent, has_reset),
    }


def openusage_meta(line):
    # Count lines are used/limit pairs; the number worth glancing at is what's
    # left (e.g. the Codex credit balance). Hide empty balances like the old
    # collector did rather than showing a "0 credits" row.
    fmt = line.get("format") if isinstance(line.get("format"), dict) else {}
    suffix = fmt.get("suffix")
    remaining = int(number_value(line, "limit")) - int(number_value(line, "used"))
    if remaining <= 0:
        return None
    value = f"{remaining} {suffix}" if isinstance(suffix, str) and suffix else f"{remaining}"
    return {
        "label": str(line.get("label") or "?"),
        "value": value,
        "tooltip": "",
        "class": "active",
    }


def quota_from_openusage(snapshot, key, name, now_epoch=None):
    now = now_epoch or time.time()
    if not isinstance(snapshot, dict):
        return quota_default(key, name, "unavailable")

    lines = [line for line in list_value(snapshot, "lines") if isinstance(line, dict)]
    # Probe failures come back as a single text line labelled "Error".
    error = next(
        (line for line in lines if line.get("type") == "text" and line.get("label") == "Error"),
        None,
    )
    if error is not None:
        return quota_default(key, name, truncate_text(str(error.get("value") or "error"), 48))

    windows = []
    meta = []
    for line in lines:
        if line.get("type") != "progress":
            continue
        fmt = line.get("format") if isinstance(line.get("format"), dict) else {}
        if fmt.get("kind") == "percent":
            windows.append(openusage_window(line, now_epoch=now))
        else:
            entry = openusage_meta(line)
            if entry is not None:
                meta.append(entry)
    if not windows and not meta:
        return quota_default(key, name, "missing")

    # Same cap as the old Antigravity collector: every window is a heavy widget,
    # so surface the models closest to their limit and keep the card short.
    if len(windows) > 4:
        windows.sort(key=lambda window: (-window["percent"], window["label"].lower()))
        windows = windows[:4]

    quota = quota_default(key, name, "live")
    quota["updated"] = time.strftime("%H:%M", time.localtime(now))
    plan = snapshot.get("plan")
    if isinstance(plan, str) and plan.strip():
        quota["plan"] = plan.strip()
    quota["windows"] = windows
    quota["meta"] = meta
    quota["class"] = quota_card_class(quota)
    return quota


def openusage_quota_states(now_epoch=None):
    report = run_text(
        ["openusage-cli", "probe"] + [key for key, _ in OPENUSAGE_PROVIDERS],
        timeout=45.0,
    )
    snapshots = parse_json(report, [])
    if not isinstance(snapshots, list):
        snapshots = []
    by_id = {
        snapshot.get("providerId"): snapshot
        for snapshot in snapshots
        if isinstance(snapshot, dict)
    }
    return [
        quota_from_openusage(by_id.get(key), key, name, now_epoch=now_epoch)
        for key, name in OPENUSAGE_PROVIDERS
    ]


CLAUDE_USAGE_URL = "https://api.anthropic.com/api/oauth/usage"
CLAUDE_OAUTH_BETA = "oauth-2025-04-20"


def claude_quota_default(status="waiting"):
    return quota_default("claude", "Claude", status)


def claude_credentials_paths():
    config_dir = os.environ.get("CLAUDE_CONFIG_DIR", "").strip()
    if config_dir:
        return [Path(config_dir).expanduser() / ".credentials.json"]
    return [
        Path("~/.claude/.credentials.json").expanduser(),
        Path("~/.config/claude/.credentials.json").expanduser(),
    ]


def claude_load_oauth():
    for path in claude_credentials_paths():
        try:
            data = json.loads(path.read_text())
        except Exception:
            continue
        oauth = data.get("claudeAiOauth") if isinstance(data, dict) else None
        if isinstance(oauth, dict) and oauth.get("accessToken"):
            return oauth
    return None


def format_claude_plan(value):
    if not isinstance(value, str) or not value.strip():
        return "--"
    return value.strip().replace("_", " ").title()


def claude_window_state(label, window, now_epoch=None):
    now = now_epoch or time.time()
    if not isinstance(window, dict):
        return {
            "label": label,
            "percent": 0,
            "value": "--",
            "remaining": "--",
            "reset": "--",
            "class": "missing",
        }

    percent = clamp_percent(number_value(window, "utilization", "used_percent", "percent"))
    reset_epoch = parse_iso_epoch(window.get("resets_at") or window.get("reset_at"))
    if reset_epoch is None:
        reset_epoch = number_value(window, "resets_at", "reset_at")

    return {
        "label": label,
        "percent": percent,
        "value": f"{percent}%",
        "remaining": format_remaining(max(0, reset_epoch - now)) if reset_epoch > 0 else "--",
        "reset": format_clock_time(reset_epoch) if reset_epoch > 0 else "--",
        "class": quota_window_class(percent, reset_epoch > 0),
    }


def claude_quota_state_from_json(usage_json, plan="--", now_epoch=None):
    body = parse_json(usage_json, {})
    if not isinstance(body, dict) or not body:
        return claude_quota_default("missing")

    now = now_epoch or time.time()
    windows = []
    for label, keys in (
        ("Session", ("five_hour", "fiveHour")),
        ("Weekly", ("seven_day", "sevenDay")),
        ("Opus weekly", ("seven_day_opus", "sevenDayOpus")),
        ("Sonnet weekly", ("seven_day_sonnet", "sevenDaySonnet")),
    ):
        window = next((body[key] for key in keys if isinstance(body.get(key), dict)), None)
        if window is not None:
            windows.append(claude_window_state(label, window, now_epoch=now))

    if not windows:
        return claude_quota_default("unrecognized data")

    quota = claude_quota_default("live")
    quota["updated"] = time.strftime("%H:%M", time.localtime(now))
    quota["plan"] = format_claude_plan(plan)
    quota["windows"] = windows
    quota["class"] = quota_card_class(quota)
    return quota


def claude_quota_state():
    oauth = claude_load_oauth()
    if not oauth:
        return claude_quota_default("not logged in")

    # Unlike the Codex collector, never refresh this token ourselves: Claude Code
    # rotates it, and a second writer racing over the refresh token can invalidate
    # the login. Re-read the file and report stale instead.
    expires_at = number_value(oauth, "expiresAt")
    if expires_at > 1e12:
        expires_at /= 1000
    if expires_at > 0 and expires_at < time.time():
        return claude_quota_default("token expired — open Claude Code")

    request = urllib.request.Request(
        CLAUDE_USAGE_URL,
        method="GET",
        headers={
            "Authorization": f"Bearer {oauth['accessToken']}",
            "anthropic-beta": CLAUDE_OAUTH_BETA,
            "Accept": "application/json",
            "User-Agent": "eww-bar",
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            usage_json = response.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        if exc.code in (401, 403):
            return claude_quota_default("unauthorized — open Claude Code")
        return claude_quota_default("unavailable")
    except Exception:
        return claude_quota_default("unavailable")

    return claude_quota_state_from_json(usage_json, plan=oauth.get("subscriptionType", "--"))


AGENT_DISPLAY_NAMES = {
    "claude": "Claude",
    "codex": "Codex",
    "gemini": "Gemini",
    "opencode": "OpenCode",
    "copilot": "Copilot",
    "amp": "Amp",
    "droid": "Droid",
}


def agent_display_name(key):
    if not isinstance(key, str) or not key:
        return "Unknown"
    return AGENT_DISPLAY_NAMES.get(key, key.replace("_", " ").title())


def period_agent_keys(row):
    keys = []
    metadata = row.get("metadata") if isinstance(row, dict) else None
    if isinstance(metadata, dict):
        keys.extend(value for value in metadata.get("agents", []) if isinstance(value, str))
    for entry in list_value(row, "agents"):
        if isinstance(entry, dict) and isinstance(entry.get("agent"), str) and entry["agent"] != "all":
            keys.append(entry["agent"])
    return [key.lower() for key in keys]


def current_period_key(kind, dict_rows, now):
    if kind == "monthly":
        return time.strftime("%Y-%m", time.localtime(now))
    if kind == "weekly":
        # Align to ccusage's own week boundaries: step forward from the most
        # recent known week start in 7-day increments until we cover `now`,
        # rather than guessing which weekday the week starts on.
        starts = [
            parse_iso_epoch(str(row.get("period") or "") + "T00:00:00")
            for row in dict_rows
        ]
        starts = [start for start in starts if start is not None]
        if starts:
            start = max(starts)
            while start + 7 * 86400 <= now:
                start += 7 * 86400
            return time.strftime("%F", time.localtime(start))
    return time.strftime("%F", time.localtime(now))


def synthetic_period_row(kind, dict_rows, now):
    # A zero-usage row keyed to the *current* period, so an idle day/week/month
    # renders as itself with no usage instead of borrowing the last active row.
    field = "period" if kind in ("weekly", "monthly") else "date"
    return {field: current_period_key(kind, dict_rows, now)}


def select_period_row(rows, kind, now_epoch=None):
    now = now_epoch or time.time()
    dict_rows = [row for row in rows if isinstance(row, dict)]
    if not dict_rows:
        return {}
    if kind == "monthly":
        current = time.strftime("%Y-%m", time.localtime(now))
        for row in dict_rows:
            if str(row.get("period") or "").startswith(current):
                return row
        return synthetic_period_row(kind, dict_rows, now)
    if kind == "weekly":
        for row in dict_rows:
            start = parse_iso_epoch(str(row.get("period") or "") + "T00:00:00")
            if start is not None and start <= now < start + 7 * 86400:
                return row
        return synthetic_period_row(kind, dict_rows, now)
    today = time.strftime("%F", time.localtime(now))
    for row in dict_rows:
        if (row.get("period") or row.get("date")) == today:
            return row
    return synthetic_period_row(kind, dict_rows, now)


def period_range_label(kind, period, now_epoch=None):
    if not period:
        return ""
    if kind == "monthly":
        try:
            return time.strftime("%B %Y", time.strptime(period, "%Y-%m"))
        except Exception:
            return period
    start = parse_iso_epoch(str(period) + "T00:00:00")
    if start is None:
        return str(period)
    if kind == "weekly":
        end = start + 6 * 86400
        return (
            f"{time.strftime('%b %-d', time.localtime(start))}"
            f" – {time.strftime('%b %-d', time.localtime(end))}"
        )
    return time.strftime("%a %b %-d", time.localtime(start))


def period_agents(row):
    total = number_value(row, "totalTokens", "total_tokens", "tokens")
    entries = list_value(row, "agents")
    if total <= 0:
        total = sum(
            number_value(entry, "totalTokens", "total_tokens", "tokens")
            for entry in entries
            if isinstance(entry, dict)
        )
    agents = []
    for entry in entries:
        if not isinstance(entry, dict):
            continue
        key = entry.get("agent")
        if not isinstance(key, str) or key == "all":
            continue
        tokens = number_value(entry, "totalTokens", "total_tokens", "tokens")
        cost = number_value(entry, "totalCost", "total_cost", "cost")
        if tokens <= 0 and cost <= 0:
            continue
        agents.append(
            {
                "key": key.lower(),
                "name": agent_display_name(key.lower()),
                "tokens": format_tokens(tokens),
                "cost": format_cost(cost),
                "percent": percent_part(tokens, total),
                "sort": tokens,
            }
        )
    agents.sort(key=lambda agent: -agent["sort"])
    for agent in agents:
        del agent["sort"]
    return agents


def period_state(rows, kind, label, now_epoch=None):
    now = now_epoch or time.time()
    row = select_period_row(rows, kind, now_epoch=now)
    values = daily_token_values(row)
    total = values["total"]
    if total <= 0:
        total = values["input"] + values["output"] + values["cache"]
    return {
        "label": label,
        "range": period_range_label(kind, row.get("period") or row.get("date") or "", now_epoch=now),
        "tokens": format_tokens(total),
        "cost": format_cost(values["cost"]),
        "agents": period_agents(row),
    }


def apply_quotas(state, quotas):
    state["quotas"] = quotas
    live = [quota for quota in quotas if quota.get("status") == "live"]
    if live and state.get("source") == "missing":
        state["class"] = "active"
        state["source"] = "quota"
        state["meta"]["status"] = "Quota only"

    # Tint the bar amber when any subscription is near its limit.
    if state.get("source") != "missing" and any(
        quota.get("class") in ("warning", "critical") for quota in live
    ):
        state["class"] = "warning"

    # Rebuild the tooltip so hovering the bar leads with the glanceable thing:
    # each live subscription's headroom, then today's local spend.
    lines = ["AI usage"]
    for quota in live:
        head = quota["name"]
        if quota.get("plan") and quota["plan"] != "--":
            head += f" · {quota['plan']}"
        for window in quota.get("windows", [])[:2]:
            if window.get("value") not in (None, "--"):
                head += f" · {window['label']} {window['value']}"
        lines.append(head)
    today = state.get("periods", {}).get("today", {})
    if today.get("tokens", "--") != "--":
        lines.append(f"Today · {today.get('tokens', '--')} / {today.get('cost', '--')}")
    if len(lines) > 1:
        state["tooltip"] = "\n".join(lines)
    return state


def ai_usage_state_from_json(report_json, now_epoch=None):
    now = now_epoch or time.time()
    report = parse_json(report_json, {})
    if not isinstance(report, dict):
        report = {}

    daily_rows = list_value(report, "daily")
    weekly_rows = list_value(report, "weekly")
    monthly_rows = list_value(report, "monthly")

    # A report is usable when ccusage returned any rows at all. A current day
    # with no recorded usage is valid data (Today = 0), not a missing collector,
    # so it must not fall through to the empty/stale placeholder.
    if not (daily_rows or weekly_rows or monthly_rows):
        return deepcopy(AI_USAGE_DEFAULT)

    today_row = select_period_row(daily_rows, "daily", now_epoch=now)

    state = deepcopy(AI_USAGE_DEFAULT)
    state["updated"] = time.strftime("%H:%M", time.localtime(now))
    state["periods"] = {
        "today": period_state(daily_rows, "daily", "Today", now_epoch=now),
        "week": period_state(weekly_rows, "weekly", "This week", now_epoch=now),
        "month": period_state(monthly_rows, "monthly", "This month", now_epoch=now),
    }
    state["text"] = f"󰙴 {state['periods']['today']['tokens']}"
    state["agents"] = agents_text(period_agent_keys(today_row))
    state["source"] = "ccusage"
    state["meta"]["status"] = "ccusage"
    state["class"] = "active"
    state["tooltip"] = (
        "AI usage\n"
        f"Today · {state['periods']['today']['tokens']} / {state['periods']['today']['cost']}"
    )
    return state


_LAST_AI_USAGE = None


def ai_usage_state():
    global _LAST_AI_USAGE

    since = time.strftime("%Y%m%d", time.localtime(time.time() - 45 * 86400))
    report = run_text(
        [
            "ccusage",
            "daily",
            "--json",
            "--offline",
            "--sections",
            "daily,weekly,monthly",
            "--by-agent",
            "--since",
            since,
        ],
        timeout=20.0,
    )
    value = ai_usage_state_from_json(report)
    apply_quotas(value, [claude_quota_state(), *openusage_quota_states()])
    if value.get("source") == "missing" and _LAST_AI_USAGE is not None:
        stale = deepcopy(_LAST_AI_USAGE)
        stale["class"] = "stale"
        stale["meta"]["stale"] = "true"
        stale["meta"]["status"] = "stale"
        stale["tooltip"] = "AI usage collector is stale\n" + stale.get("tooltip", "")
        return stale
    if value.get("source") != "missing":
        _LAST_AI_USAGE = deepcopy(value)
    return value


def refresh_ai_usage(state):
    # Flag the refresh immediately so the popup can show a loading indicator over
    # the existing (still valid) data rather than blanking or sitting silent while
    # the combined ccusage + quota fetch runs in the background.
    current = state.get("ai_usage")
    pending = deepcopy(current) if isinstance(current, dict) else deepcopy(AI_USAGE_DEFAULT)
    meta = pending.get("meta") if isinstance(pending.get("meta"), dict) else {}
    if meta.get("refreshing") != "true":
        pending["meta"] = {**meta, "refreshing": "true"}
        state.update(ai_usage=pending)

    try:
        fresh = ai_usage_state()
        fresh["meta"]["refreshing"] = "false"
    except Exception:
        # Never leave the popup indicator pulsing on a transient failure; keep the
        # data we already had and just clear the flag.
        fresh = deepcopy(pending)
        fresh["meta"] = {**fresh.get("meta", {}), "refreshing": "false"}
    state.update(ai_usage=fresh)
    return fresh


def cpu_state():
    def read_cpu():
        parts = Path("/proc/stat").read_text().splitlines()[0].split()
        values = [int(value) for value in parts[1:8]]
        idle = values[3] + values[4]
        total = sum(values)
        return total, idle

    try:
        total1, idle1 = read_cpu()
        time.sleep(0.2)
        total2, idle2 = read_cpu()
        total_delta = total2 - total1
        idle_delta = idle2 - idle1
        if total_delta <= 0:
            return " --%"
        usage = (total_delta - idle_delta) * 100 / total_delta
        return f" {usage:.0f}%"
    except Exception:
        return " --%"


def memory_state_from_text(text):
    values = {}
    for line in text.splitlines():
        parts = line.split()
        if len(parts) >= 2:
            values[parts[0].rstrip(":")] = int(parts[1])

    total = values.get("MemTotal", 0)
    available = values.get("MemAvailable", 0)
    swap_total = values.get("SwapTotal", 0)
    swap_free = values.get("SwapFree", 0)
    if total <= 0:
        return {"text": " --%", "tooltip": "", "class": ""}

    pct = (total - available) / total * 100
    used_g = (total - available) / 1048576
    total_g = total / 1048576
    swap_used_g = (swap_total - swap_free) / 1048576
    swap_total_g = swap_total / 1048576
    cls = "critical" if pct >= 80 else ""
    return {
        "text": f" {pct:.0f}%",
        "tooltip": f"Used: {used_g:.1f}G/{total_g:.1f}G\nSwap: {swap_used_g:.1f}G/{swap_total_g:.1f}G",
        "class": cls,
    }


def memory_state():
    try:
        return memory_state_from_text(Path("/proc/meminfo").read_text())
    except Exception:
        return {"text": " --%", "tooltip": "", "class": ""}


def temperature_state():
    max_temp = None
    for input_path in Path("/sys/class/hwmon").glob("hwmon*/temp*_input"):
        try:
            temp = int(input_path.read_text().strip()) / 1000
        except Exception:
            continue
        if max_temp is None or temp > max_temp:
            max_temp = temp

    if max_temp is None:
        return {"text": " --°C", "class": ""}
    rounded = int(round(max_temp))
    return {
        "text": f" {rounded}°C",
        "class": "critical" if rounded >= 90 else "",
    }


def first_ip(ip_text):
    for line in ip_text.splitlines():
        if ":" in line:
            return line.split(":", 1)[1]
    return ""


def nmcli_value(text, key):
    for line in text.splitlines():
        if line.startswith(key + ":"):
            return line.split(":", 1)[1]
    return ""


def wireless_signal_percent(wireless_text, interface):
    for line in wireless_text.splitlines():
        stripped = line.strip()
        if not stripped.startswith(interface + ":"):
            continue
        parts = stripped.replace(":", " ").split()
        if len(parts) < 3:
            return None
        try:
            quality = float(parts[2].rstrip("."))
        except ValueError:
            return None
        return max(0, min(100, int(round(quality * 100 / 70))))
    return None


def connected_device(status_text, device_type):
    for line in status_text.splitlines():
        fields = line.split(":")
        if len(fields) >= 3 and fields[1] == device_type and fields[2] == "connected":
            return fields[0]
    return ""


def network_state_from_text(status_text, wifi_text, ip_by_device):
    wifi_device = ""
    for line in status_text.splitlines():
        fields = line.split(":")
        if len(fields) >= 3 and fields[1] == "wifi" and fields[2] == "connected":
            wifi_device = fields[0]
            break

    wifi = ""
    for line in wifi_text.splitlines():
        fields = line.split(":")
        if len(fields) >= 3 and fields[0] == "yes":
            wifi = line
            break

    if wifi_device and wifi:
        fields = wifi.split(":")
        ssid = fields[1]
        signal = fields[2]
        ip_info = first_ip(ip_by_device.get(wifi_device, ""))
        if ip_info:
            return {
                "text": f" {ssid} {signal}%",
                "tooltip": f"{wifi_device}: {ip_info}",
                "class": "wifi",
            }
        return {
            "text": "󰈀 (No IP)",
            "tooltip": f"{wifi_device}: No IP",
            "class": "linked",
        }

    for line in status_text.splitlines():
        fields = line.split(":")
        if len(fields) >= 3 and fields[1] == "ethernet" and fields[2] == "connected":
            device = fields[0]
            ip_info = first_ip(ip_by_device.get(device, ""))
            if ip_info:
                return {
                    "text": f"󰌗 {device}",
                    "tooltip": f"{device}: {ip_info}",
                    "class": "ethernet",
                }
            return {
                "text": "󰈀 (No IP)",
                "tooltip": f"{device}: No IP",
                "class": "linked",
            }

    return {
        "text": "⚠ Disconnected",
        "tooltip": "No connection",
        "class": "disconnected",
    }


def default_route_device(route_text):
    """Interface owning the default route, lowest metric winning.

    A docked laptop carries one default route per link, and the kernel prefers
    the lowest metric; an absent metric means 0. ECMP routes (a bare `default`
    line followed by `nexthop` lines) are deliberately not handled - they have
    no single owning interface to name.
    """
    best_device = ""
    best_metric = None
    for line in route_text.splitlines():
        fields = line.split()
        if not fields or fields[0] != "default" or "dev" not in fields:
            continue
        device_index = fields.index("dev") + 1
        if device_index >= len(fields):
            continue
        metric = 0
        if "metric" in fields:
            metric_index = fields.index("metric") + 1
            if metric_index < len(fields):
                try:
                    metric = int(fields[metric_index])
                except ValueError:
                    metric = 0
        if best_metric is None or metric < best_metric:
            best_device = fields[device_index]
            best_metric = metric
    return best_device


def device_ipv4(addr_text, device):
    # `ip -o -4 addr show` lines look like:
    #   2: eno1    inet 192.168.0.88/24 brd ... scope global dynamic eno1
    for line in addr_text.splitlines():
        fields = line.split()
        if len(fields) >= 4 and fields[1] == device and fields[2] == "inet":
            return fields[3].split("/")[0]
    return ""


def link_state_from_text(route_text, addr_text):
    """Readout for when nmcli reports nothing usable.

    The kernel keeps the lease and the default route after NetworkManager
    stops, so the machine is still online; only the usual source of truth is
    gone. The same holds when NetworkManager is up but owns nothing, e.g.
    unmanaged devices or a route belonging to systemd-networkd or a tunnel.
    Report the interface that owns the route rather than claiming to be
    disconnected.

    run_text() flattens "not running", "not on PATH" and "timed out" into the
    same empty string, so the tooltip names where the answer came from instead
    of asserting a cause it cannot know.
    """
    device = default_route_device(route_text)
    if not device:
        return None
    ip_info = device_ipv4(addr_text, device)
    return {
        "text": f"󰌗 {device} ⚠",
        "tooltip": (
            f"{device}: {ip_info or 'No IP'} "
            "(kernel routing table; NetworkManager not reporting)"
        ),
        "class": "degraded",
    }


def link_fallback_state():
    return link_state_from_text(
        run_text(["ip", "route"]),
        run_text(["ip", "-o", "-4", "addr", "show"]),
    )


def network_connection_state():
    status = run_text(["nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status"])
    wifi_device = connected_device(status, "wifi")
    ethernet_device = connected_device(status, "ethernet")
    # Whichever interface carries the default route is the one actually in use.
    if ethernet_device and default_route_device(run_text(["ip", "route"])) == ethernet_device:
        wifi_device = ""
    if wifi_device:
        details = run_text(
            ["nmcli", "-t", "-f", "GENERAL.CONNECTION,IP4.ADDRESS", "dev", "show", wifi_device]
        )
        ssid = nmcli_value(details, "GENERAL.CONNECTION") or wifi_device
        ip_info = first_ip(details)
        try:
            wireless_text = Path("/proc/net/wireless").read_text()
        except Exception:
            wireless_text = ""
        signal = wireless_signal_percent(wireless_text, wifi_device)
        if signal is None:
            wifi = run_text(["nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL", "dev", "wifi"])
            return network_state_from_text(status, wifi, {wifi_device: details})
        if ip_info:
            return {
                "text": f" {ssid} {signal}%",
                "tooltip": f"{wifi_device}: {ip_info}",
                "class": "wifi",
            }
        return {
            "text": "󰈀 (No IP)",
            "tooltip": f"{wifi_device}: No IP",
            "class": "linked",
        }

    if ethernet_device:
        details = run_text(["nmcli", "-t", "-f", "IP4.ADDRESS", "dev", "show", ethernet_device])
        ip_info = first_ip(details)
        if ip_info:
            return {
                "text": f"󰌗 {ethernet_device}",
                "tooltip": f"{ethernet_device}: {ip_info}",
                "class": "ethernet",
            }
        return {
            "text": "󰈀 (No IP)",
            "tooltip": f"{ethernet_device}: No IP",
            "class": "linked",
        }

    # nmcli claims nothing is connected. Before believing it, check whether the
    # kernel still has a default route: NetworkManager may be stopped, absent,
    # or simply not the owner of this machine's networking.
    fallback = link_fallback_state()
    if fallback:
        return fallback

    return {
        "text": "⚠ Disconnected",
        "tooltip": "No connection",
        "class": "disconnected",
    }


def network_radio_enabled():
    return "true" if run_text(["nmcli", "radio", "wifi"]).strip() == "enabled" else "false"


def network_state():
    result = network_connection_state()
    result["wifi_enabled"] = network_radio_enabled()
    return result


def volume_label_from_text(text):
    # The bar label string. Kept byte-for-byte compatible with the previous
    # behavior (same speaker glyphs and thresholds).
    match = re.search(r"Volume:\s+([0-9.]+)", text)
    if not match:
        return ""
    volume = int((Decimal(match.group(1)) * 100).to_integral_value(rounding=ROUND_HALF_UP))
    if "[MUTED]" in text:
        return "󰝟"
    if volume < 35:
        return f" {volume}%"
    if volume < 70:
        return f" {volume}%"
    return f" {volume}%"


def sinks_from_pactl_json(json_text, default_name):
    data = parse_json(json_text, [])
    sinks = []
    if isinstance(data, list):
        for entry in data:
            if not isinstance(entry, dict):
                continue
            name = entry.get("name")
            if not isinstance(name, str) or not name:
                continue
            description = entry.get("description")
            sinks.append(
                {
                    "name": name,
                    "description": truncate_text(
                        description if isinstance(description, str) and description else name, 30
                    ),
                    "active": "true" if name == default_name else "false",
                }
            )
    return sinks


def sinks_from_pactl_short(short_text, default_name):
    sinks = []
    for line in short_text.splitlines():
        fields = line.split("\t")
        if len(fields) < 2 or not fields[1]:
            continue
        name = fields[1]
        sinks.append(
            {
                "name": name,
                "description": truncate_text(name, 30),
                "active": "true" if name == default_name else "false",
            }
        )
    return sinks


def volume_sinks():
    default_name = run_text(["pactl", "get-default-sink"]).strip()
    sinks = sinks_from_pactl_json(run_text(["pactl", "-f", "json", "list", "sinks"]), default_name)
    if not sinks:
        sinks = sinks_from_pactl_short(run_text(["pactl", "list", "short", "sinks"]), default_name)
    return sinks


def volume_state_from_text(text, sinks=None):
    sinks = sinks or []
    label = volume_label_from_text(text)
    match = re.search(r"Volume:\s+([0-9.]+)", text)
    if not match:
        return dict(VOLUME_DEFAULT, sinks=sinks)
    percent = int((Decimal(match.group(1)) * 100).to_integral_value(rounding=ROUND_HALF_UP))
    muted = "[MUTED]" in text
    return {
        "text": label,
        "percent": percent,
        "muted": "true" if muted else "false",
        "class": "muted" if muted else "",
        "sinks": sinks,
    }


def volume_state():
    return volume_state_from_text(
        run_text(["wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"]),
        sinks=volume_sinks(),
    )


def tray_count_from_text(text):
    # `busctl get-property ... RegisteredStatusNotifierItems` prints the value as
    # `as N "item1" "item2" ...` — the token after the `as` type tag is the count.
    parts = text.split()
    if len(parts) >= 2 and parts[0] == "as":
        try:
            return int(parts[1])
        except ValueError:
            return 0
    return 0


def tray_count():
    return tray_count_from_text(
        run_text(
            [
                "busctl", "--user", "get-property",
                "org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher",
                "org.kde.StatusNotifierWatcher", "RegisteredStatusNotifierItems",
            ]
        )
    )


def battery_state(root=Path("/sys/class/power_supply")):
    for battery in root.glob("BAT*"):
        capacity_path = battery / "capacity"
        if not capacity_path.exists():
            continue
        try:
            capacity = int(capacity_path.read_text().strip())
        except Exception:
            continue
        try:
            status = (battery / "status").read_text().strip()
        except Exception:
            status = ""

        if status == "Charging":
            icon = "󰂄"
        elif capacity < 20:
            icon = "󰁻"
        elif capacity < 40:
            icon = "󰁼"
        elif capacity < 60:
            icon = "󰁾"
        elif capacity < 80:
            icon = "󰂀"
        else:
            icon = "󰁹"

        now = rate = full_value = design_value = voltage = None
        unit = None
        for unit_name, now_name, rate_name, full_name, design_name in [
            ("energy", "energy_now", "power_now", "energy_full", "energy_full_design"),
            ("charge", "charge_now", "current_now", "charge_full", "charge_full_design"),
        ]:
            try:
                now = int((battery / now_name).read_text().strip())
                rate = int((battery / rate_name).read_text().strip())
                full_value = int((battery / full_name).read_text().strip())
                unit = unit_name
            except Exception:
                now = rate = full_value = None
                continue
            try:
                design_value = int((battery / design_name).read_text().strip())
            except Exception:
                design_value = None
            if unit == "charge":
                try:
                    voltage = int((battery / "voltage_now").read_text().strip())
                except Exception:
                    voltage = None
            break

        time_str = "N/A"
        minutes = None
        if now is not None and rate and rate > 0:
            if status == "Discharging":
                minutes = now * 60 // rate
            elif status == "Charging" and full_value:
                minutes = (full_value - now) * 60 // rate
        if minutes is not None:
            time_str = f"{minutes // 60}h {minutes % 60}m"

        health = "--"
        if full_value and design_value and design_value > 0:
            health = f"{int(round(full_value * 100 / design_value))}%"

        power = "—"
        if rate and rate > 0:
            if unit == "energy":
                watts = rate / 1_000_000
            elif unit == "charge" and voltage:
                watts = rate * voltage / 1_000_000_000_000
            else:
                watts = None
            if watts is not None:
                power = f"{watts:.1f} W"

        cls = ""
        if status == "Charging":
            cls = "charging"
        elif capacity < 20:
            cls = "critical"
        elif capacity < 30:
            cls = "warning"
        return {
            "text": f"{icon} {capacity}%",
            "alt": f"{icon} {time_str}",
            "capacity": capacity,
            "class": cls,
            "status": status or "Unknown",
            "time": time_str,
            "health": health,
            "power": power,
        }
    return BATTERY_DEFAULT.copy()


def parse_controller(controller_text):
    address = ""
    alias = ""
    powered = "off"
    for line in controller_text.splitlines():
        stripped = line.strip()
        if stripped.startswith("Controller "):
            parts = stripped.split()
            if len(parts) >= 2:
                address = parts[1]
        elif stripped.startswith("Alias:"):
            alias = stripped.split(":", 1)[1].strip()
        elif stripped.startswith("Powered:"):
            powered = stripped.split(":", 1)[1].strip()
    return alias or "Bluetooth", address or "N/A", powered


def parse_device_info(info_text, fallback_alias):
    alias = fallback_alias
    battery = ""
    for line in info_text.splitlines():
        stripped = line.strip()
        if stripped.startswith("Alias:"):
            alias = stripped.split(":", 1)[1].strip()
        match = re.search(r"Battery Percentage:.*\((\d+)\)", stripped)
        if match:
            battery = match.group(1) + "%"
    return alias, battery


def bluetooth_state_from_text(controller_text, devices_text, info_by_address):
    controller_alias, controller_address, powered = parse_controller(controller_text)
    powered_flag = "true" if powered == "yes" else "false"
    devices = [line for line in devices_text.splitlines() if line.strip()]

    structured = []
    for line in devices:
        parts = line.split(maxsplit=2)
        if len(parts) < 2:
            continue
        address = parts[1]
        fallback = parts[2] if len(parts) >= 3 else address
        alias, battery = parse_device_info(info_by_address.get(address, ""), fallback)
        structured.append(
            {"mac": address, "name": truncate_text(alias, 24), "battery": battery, "connected": "true"}
        )

    if not devices:
        tooltip = f"{controller_alias}\t{controller_address}\n\n0 connected"
        return {"text": "", "tooltip": tooltip, "class": powered, "powered": powered_flag, "devices": []}

    preferred = next(
        (line for line in devices if re.search(r"ugreen_1|ugreen_2", line, re.I)),
        devices[0],
    )
    preferred_parts = preferred.split(maxsplit=2)
    first_address = preferred_parts[1] if len(preferred_parts) >= 2 else ""
    fallback_alias = preferred_parts[2] if len(preferred_parts) >= 3 else first_address
    first_alias, first_battery = parse_device_info(
        info_by_address.get(first_address, ""), fallback_alias
    )
    text = f"󰂯 {first_alias} {first_battery}" if first_battery else f"󰂯 {first_alias}"
    text = truncate_text(text, 24)

    detail_lines = []
    for device in structured:
        if device["battery"]:
            detail_lines.append(f"{device['name']}\t{device['mac']}\t{device['battery']}")
        else:
            detail_lines.append(f"{device['name']}\t{device['mac']}")

    tooltip = (
        f"{controller_alias}\t{controller_address}\n\n"
        f"{len(devices)} connected\n\n" + "\n".join(detail_lines)
    )
    return {
        "text": text,
        "tooltip": tooltip,
        "class": "connected",
        "powered": powered_flag,
        "devices": structured,
    }


def bluetooth_state():
    controller = run_text(["bluetoothctl", "show"])
    devices = run_text(["bluetoothctl", "devices", "Connected"])
    info_by_address = {}
    for line in devices.splitlines():
        parts = line.split(maxsplit=2)
        if len(parts) >= 2:
            info_by_address[parts[1]] = run_text(["bluetoothctl", "info", parts[1]])
    return bluetooth_state_from_text(controller, devices, info_by_address)


def idle_inhibited_state():
    pidfile = idle_pidfile_path()
    try:
        pid = int(pidfile.read_text().strip())
    except Exception:
        return "false"
    return "true" if Path(f"/proc/{pid}").exists() else "false"
