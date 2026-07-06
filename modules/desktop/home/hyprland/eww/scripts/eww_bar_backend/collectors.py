import base64
import json
import os
import re
import time
import urllib.error
import urllib.parse
import urllib.request
from copy import deepcopy
from datetime import datetime
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path

from .common import (
    AI_USAGE_DEFAULT,
    BATTERY_DEFAULT,
    CCUSAGE_DEFAULT,
    MEDIA_DEFAULT,
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
        "text": truncate_text(metadata[0].strip(), 52),
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


def nested_number(data, *paths):
    for path in paths:
        current = data
        for key in path:
            if not isinstance(current, dict):
                current = None
                break
            current = current.get(key)
        if isinstance(current, bool):
            continue
        if isinstance(current, (int, float)):
            return current
        if isinstance(current, str):
            try:
                return float(current)
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
        return f"{value / 1_000_000:.1f}m"
    if value >= 1_000:
        return f"{value / 1_000:.1f}k"
    return f"{int(value)}"


def format_cost(value):
    if value <= 0:
        return "--"
    return f"${value:.2f}"


def format_burn_rate(value):
    if value <= 0:
        return "--"
    return f"${value:.2f}/h"


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


def jwt_payload(token):
    if not isinstance(token, str):
        return {}
    parts = token.split(".")
    if len(parts) < 2:
        return {}
    payload = parts[1]
    payload += "=" * (-len(payload) % 4)
    try:
        data = base64.urlsafe_b64decode(payload.encode("ascii"))
        value = json.loads(data.decode("utf-8"))
    except Exception:
        return {}
    return value if isinstance(value, dict) else {}


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


def ccusage_state_from_json(daily_json, blocks_json, now_epoch=None):
    daily_report = parse_json(daily_json, {})
    blocks_report = parse_json(blocks_json, {})
    if not daily_report and not blocks_report:
        return CCUSAGE_DEFAULT.copy()

    token_values = daily_token_values(daily_row_from_report(daily_report, now_epoch=now_epoch))
    input_tokens = token_values["input"]
    output_tokens = token_values["output"]
    cache_tokens = token_values["cache"]
    total_tokens = token_values["total"]
    total_cost = token_values["cost"]

    active_blocks = [
        row
        for row in list_value(blocks_report, "blocks", "data")
        if isinstance(row, dict) and row.get("isActive") is True
    ]
    active_block = active_blocks[0] if active_blocks else {}
    block_tokens = number_value(active_block, "totalTokens", "total_tokens", "tokens")
    block_cost = number_value(active_block, "totalCost", "total_cost", "costUSD", "cost")
    projected_tokens = nested_number(
        active_block,
        ("projection", "totalTokens"),
        ("projection", "total_tokens"),
        ("projected", "totalTokens"),
    )
    projected_cost = nested_number(
        active_block,
        ("projection", "totalCost"),
        ("projection", "total_cost"),
        ("projection", "costUSD"),
    )
    projected_remaining_minutes = nested_number(
        active_block,
        ("projection", "remainingMinutes"),
        ("projection", "remaining_minutes"),
    )

    now = now_epoch or time.time()
    start = parse_iso_epoch(active_block.get("startTime") or active_block.get("start_time"))
    end = parse_iso_epoch(active_block.get("endTime") or active_block.get("end_time"))
    if start is not None and end is not None and end > start:
        percent = max(0, min(100, int(round((now - start) * 100 / (end - start)))))
        if projected_remaining_minutes > 0:
            remaining = format_remaining(projected_remaining_minutes * 60)
        else:
            remaining = format_remaining(end - now)
    else:
        percent = 0
        remaining = (
            format_remaining(projected_remaining_minutes * 60)
            if projected_remaining_minutes > 0
            else "--"
        )

    if total_tokens <= 0 and block_tokens <= 0:
        return CCUSAGE_DEFAULT.copy()

    cls = "active" if active_block else ""
    if total_cost >= 20:
        cls = "warning"
    if percent >= 85:
        cls = "warning"

    today_tokens = format_tokens(total_tokens)
    today_cost = format_cost(total_cost)
    block_text = format_tokens(block_tokens)
    projected_text = format_tokens(projected_tokens)
    block_cost_text = format_cost(block_cost)
    projected_cost_text = format_cost(projected_cost)
    tooltip = (
        f"Today: {today_tokens} tokens / {today_cost}\n"
        f"Input: {format_tokens(input_tokens)}  Output: {format_tokens(output_tokens)}  Cache: {format_tokens(cache_tokens)}\n"
        f"Active block: {block_text} used, {projected_text} projected\n"
        f"Remaining: {remaining}"
    )

    return {
        "text": f"󱃖 {today_tokens}",
        "tooltip": tooltip,
        "class": cls,
        "updated": time.strftime("%H:%M", time.localtime(now)),
        "today": {
            "tokens": today_tokens,
            "cost": today_cost,
            "input": format_tokens(input_tokens),
            "output": format_tokens(output_tokens),
            "cache": format_tokens(cache_tokens),
        },
        "block": {
            "tokens": block_text,
            "projected": projected_text,
            "percent": percent,
            "remaining": remaining,
            "cost": block_cost_text,
            "projected_cost": projected_cost_text,
        },
    }


def ccusage_state():
    today = time.strftime("%F")
    daily = run_text(
        [
            "ccusage",
            "daily",
            "--json",
            "--offline",
            "--since",
            today,
            "--until",
            today,
        ],
        timeout=15.0,
    )
    blocks = run_text(["ccusage", "blocks", "--json", "--offline"], timeout=15.0)
    return ccusage_state_from_json(daily, blocks)


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


def provider_cards_from_agents(agents):
    providers = deepcopy(AI_USAGE_DEFAULT["providers"])
    aliases = {
        "claude": ("claude", "Claude Code"),
        "codex": ("codex", "Codex CLI"),
        "gemini": ("gemini", "Gemini CLI"),
    }
    for token in agents:
        for key, (_prefix, label) in aliases.items():
            if key in token:
                providers[key]["class"] = "active"
                providers[key]["status"] = "Local usage"
                providers[key]["detail"] = label
    return providers


def openusage_block_from_json(blocks_json, now_epoch=None):
    report = parse_json(blocks_json, {})
    rows = list_value(report, "rows", "blocks", "data")
    active_row = next(
        (row for row in rows if isinstance(row, dict) and row.get("active") is True),
        None,
    )
    if active_row is None and rows:
        active_row = next((row for row in reversed(rows) if isinstance(row, dict)), None)
    if not active_row:
        return None

    tokens = number_value(active_row, "total_tokens", "totalTokens", "tokens")
    cost = number_value(active_row, "cost_usd", "costUSD", "totalCost", "cost")
    projected_cost = number_value(active_row, "projected_cost_usd", "projectedCostUSD")
    burn_rate = number_value(active_row, "burn_rate_usd_per_hour", "burnRateUSDPerHour")
    remaining_seconds = number_value(active_row, "time_remaining_seconds", "timeRemainingSeconds")

    now = now_epoch or time.time()
    start = parse_iso_epoch(active_row.get("start") or active_row.get("startTime"))
    end = parse_iso_epoch(active_row.get("end") or active_row.get("endTime"))
    if start is not None and end is not None and end > start:
        percent = max(0, min(100, int(round((now - start) * 100 / (end - start)))))
        remaining = format_remaining(remaining_seconds if remaining_seconds > 0 else end - now)
    else:
        percent = 0
        remaining = format_remaining(remaining_seconds) if remaining_seconds > 0 else "--"

    return {
        "tokens": format_tokens(tokens),
        "projected": "--",
        "percent": percent,
        "remaining": remaining,
        "cost": format_cost(cost),
        "projected_cost": format_cost(projected_cost),
        "burn_rate": format_burn_rate(burn_rate),
        "source": "OpenUsage",
        "label": active_row.get("label") or "Active block",
    }


CODEX_CLIENT_ID = "app_EMoamEEZ73f0CkXaXp7hrann"
CODEX_USAGE_URL = "https://chatgpt.com/backend-api/wham/usage"
CODEX_REFRESH_URL = "https://auth.openai.com/oauth/token"
CODEX_RESET_CREDITS_URL = "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits"


def codex_subscription_default(status="waiting"):
    state = deepcopy(AI_USAGE_DEFAULT["subscription"])
    state["status"] = status
    return state


def codex_auth_paths():
    codex_home = os.environ.get("CODEX_HOME", "").strip()
    if codex_home:
        return [Path(codex_home).expanduser() / "auth.json"]
    return [
        Path("~/.config/codex/auth.json").expanduser(),
        Path("~/.codex/auth.json").expanduser(),
    ]


def codex_load_auth():
    for path in codex_auth_paths():
        try:
            data = json.loads(path.read_text())
        except Exception:
            continue
        if isinstance(data, dict) and isinstance(data.get("tokens"), dict):
            return data, path
    return None, None


def codex_save_auth(auth, path):
    if path is None:
        return
    try:
        path.write_text(json.dumps(auth, indent=2, sort_keys=True) + "\n")
    except Exception:
        pass


def codex_token_expires_at(access_token):
    payload = jwt_payload(access_token)
    exp = number_value(payload, "exp")
    return exp if exp > 0 else None


def codex_needs_refresh(auth, now_epoch=None):
    tokens = auth.get("tokens") if isinstance(auth, dict) else {}
    if not isinstance(tokens, dict):
        return False
    expires_at = codex_token_expires_at(tokens.get("access_token"))
    if expires_at is None:
        return False
    return expires_at - (now_epoch or time.time()) <= 300


def codex_refresh_auth(auth, path):
    tokens = auth.get("tokens") if isinstance(auth, dict) else {}
    if not isinstance(tokens, dict):
        return auth
    refresh_token = tokens.get("refresh_token")
    if not refresh_token:
        return auth

    body = urllib.parse.urlencode(
        {
            "grant_type": "refresh_token",
            "client_id": CODEX_CLIENT_ID,
            "refresh_token": refresh_token,
        }
    ).encode("utf-8")
    request = urllib.request.Request(
        CODEX_REFRESH_URL,
        data=body,
        method="POST",
        headers={"Content-Type": "application/x-www-form-urlencoded"},
    )
    with urllib.request.urlopen(request, timeout=15) as response:
        payload = json.loads(response.read().decode("utf-8"))

    access_token = payload.get("access_token")
    if not access_token:
        return auth
    tokens["access_token"] = access_token
    if payload.get("refresh_token"):
        tokens["refresh_token"] = payload["refresh_token"]
    if payload.get("id_token"):
        tokens["id_token"] = payload["id_token"]
    auth["tokens"] = tokens
    auth["last_refresh"] = datetime.utcnow().replace(microsecond=0).isoformat() + "Z"
    codex_save_auth(auth, path)
    return auth


def codex_request_json(url, auth, timeout=10.0, extra_headers=None):
    tokens = auth.get("tokens") if isinstance(auth, dict) else {}
    if not isinstance(tokens, dict):
        return ""
    access_token = tokens.get("access_token")
    if not access_token:
        return ""
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Accept": "application/json",
        "User-Agent": "eww-bar",
    }
    account_id = tokens.get("account_id")
    if account_id:
        headers["ChatGPT-Account-Id"] = account_id
    if extra_headers:
        headers.update(extra_headers)
    request = urllib.request.Request(url, method="GET", headers=headers)
    with urllib.request.urlopen(request, timeout=timeout) as response:
        return response.read().decode("utf-8")


def codex_fetch_usage_json(auth, path):
    try:
        return codex_request_json(CODEX_USAGE_URL, auth, timeout=10.0)
    except urllib.error.HTTPError as exc:
        if exc.code not in (401, 403):
            raise
    auth = codex_refresh_auth(auth, path)
    return codex_request_json(CODEX_USAGE_URL, auth, timeout=10.0)


def codex_fetch_reset_credits_json(auth):
    return codex_request_json(
        CODEX_RESET_CREDITS_URL,
        auth,
        timeout=6.0,
        extra_headers={
            "OpenAI-Beta": "codex-1",
            "originator": "Codex Desktop",
        },
    )


def format_codex_plan(value):
    if not isinstance(value, str) or not value.strip():
        return "--"
    raw = value.strip()
    lowered = raw.lower()
    if lowered == "prolite":
        return "Pro 5x"
    if lowered == "pro":
        return "Pro 20x"
    return raw.replace("_", " ").title()


def codex_window_state(label, window, now_epoch=None):
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

    percent = clamp_percent(number_value(window, "used_percent"))
    reset_epoch = number_value(window, "reset_at")
    if reset_epoch <= 0:
        reset_after = number_value(window, "reset_after_seconds")
        reset_epoch = now + reset_after if reset_after > 0 else 0
    period_seconds = number_value(window, "limit_window_seconds")
    if period_seconds > 0 and reset_epoch > 0:
        remaining_to_reset = reset_epoch - now
        if remaining_to_reset >= period_seconds - 30 and percent <= 1:
            percent = 0

    cls = "missing"
    if percent >= 90:
        cls = "critical"
    elif percent >= 80:
        cls = "warning"
    elif percent > 0:
        cls = "active"
    elif reset_epoch > 0:
        cls = "empty"

    return {
        "label": label,
        "percent": percent,
        "value": f"{percent}%",
        "remaining": format_remaining(max(0, reset_epoch - now)) if reset_epoch > 0 else "--",
        "reset": format_clock_time(reset_epoch) if reset_epoch > 0 else "--",
        "class": cls,
    }


def codex_reset_credits_state(usage_body, reset_credits_body, now_epoch=None):
    now = now_epoch or time.time()
    source = (
        reset_credits_body
        if isinstance(reset_credits_body, dict) and "available_count" in reset_credits_body
        else None
    )
    if source is None:
        source = usage_body.get("rate_limit_reset_credits") if isinstance(usage_body, dict) else None
    if not isinstance(source, dict):
        return {"text": "--", "tooltip": "", "class": "missing"}

    count = max(0, int(number_value(source, "available_count")))
    expiries = []
    credits = source.get("credits")
    if isinstance(credits, list):
        for credit in credits:
            if not isinstance(credit, dict):
                continue
            status = credit.get("status")
            if isinstance(status, str) and status != "available":
                continue
            expires = parse_iso_epoch(credit.get("expires_at"))
            if expires is None:
                expires = number_value(credit, "expires_at")
            if expires > 0:
                expiries.append(expires)
    expiries.sort()

    cls = "active" if count > 0 else "missing"
    if expiries and expiries[0] - now <= 24 * 60 * 60:
        cls = "warning"
    tooltip = "\n".join(format_clock_time(expiry) for expiry in expiries)
    return {
        "text": f"{count} available",
        "tooltip": tooltip,
        "class": cls,
    }


def codex_credits_state(usage_body):
    credits = usage_body.get("credits") if isinstance(usage_body, dict) else None
    balance = number_value(credits, "balance") if isinstance(credits, dict) else 0
    if balance <= 0:
        return {"text": "--", "class": "missing"}
    count = max(0, int(balance))
    return {
        "text": f"${count * 0.04:.2f} · {count} credits",
        "class": "active",
    }


def codex_subscription_class(subscription):
    classes = [
        subscription.get("session", {}).get("class"),
        subscription.get("weekly", {}).get("class"),
        subscription.get("spark", {}).get("class"),
        subscription.get("spark_weekly", {}).get("class"),
        subscription.get("resets", {}).get("class"),
    ]
    for name in ("critical", "warning", "active", "empty"):
        if name in classes:
            return name
    return "missing"


def codex_subscription_state_from_json(
    usage_json,
    reset_credits_json="",
    now_epoch=None,
):
    body = parse_json(usage_json, {})
    if not isinstance(body, dict) or not body:
        return codex_subscription_default("missing")
    reset_body = parse_json(reset_credits_json, {})
    if not isinstance(reset_body, dict):
        reset_body = {}

    now = now_epoch or time.time()
    rate_limit = body.get("rate_limit") if isinstance(body.get("rate_limit"), dict) else {}
    state = codex_subscription_default("live")
    state["source"] = "codex"
    state["updated"] = time.strftime("%H:%M", time.localtime(now))
    state["plan"] = format_codex_plan(body.get("plan_type"))
    state["session"] = codex_window_state("Session", rate_limit.get("primary_window"), now_epoch=now)
    state["weekly"] = codex_window_state("Weekly", rate_limit.get("secondary_window"), now_epoch=now)

    for entry in list_value(body, "additional_rate_limits"):
        if not isinstance(entry, dict):
            continue
        names = [
            entry.get("limit_name", ""),
            entry.get("metered_feature", ""),
        ]
        if not any(isinstance(name, str) and "spark" in name.lower() for name in names):
            continue
        spark_limit = entry.get("rate_limit") if isinstance(entry.get("rate_limit"), dict) else {}
        state["spark"] = codex_window_state("Spark", spark_limit.get("primary_window"), now_epoch=now)
        state["spark_weekly"] = codex_window_state("Spark Weekly", spark_limit.get("secondary_window"), now_epoch=now)
        break

    state["resets"] = codex_reset_credits_state(body, reset_body, now_epoch=now)
    state["credits"] = codex_credits_state(body)
    state["class"] = codex_subscription_class(state)
    return state


def codex_subscription_state():
    auth, path = codex_load_auth()
    if not auth:
        return codex_subscription_default("not logged in")
    tokens = auth.get("tokens") if isinstance(auth, dict) else {}
    if not isinstance(tokens, dict) or not tokens.get("access_token"):
        return codex_subscription_default("api-key only")

    try:
        if codex_needs_refresh(auth):
            auth = codex_refresh_auth(auth, path)
        usage_json = codex_fetch_usage_json(auth, path)
        try:
            reset_credits_json = codex_fetch_reset_credits_json(auth)
        except Exception:
            reset_credits_json = ""
        return codex_subscription_state_from_json(usage_json, reset_credits_json)
    except Exception:
        return codex_subscription_default("unavailable")


def apply_codex_subscription(state, subscription):
    state["subscription"] = subscription
    if subscription.get("status") != "live":
        return state

    plan = subscription.get("plan") or "--"
    session = subscription.get("session", {})
    session_value = session.get("value", "--")
    provider = state["providers"].get("codex", {})
    provider["class"] = "active"
    provider["status"] = "Subscription"
    provider["detail"] = f"{plan} · Session {session_value}"
    state["providers"]["codex"] = provider
    if state.get("source") == "missing":
        state["class"] = "active"
        state["source"] = "codex"
        state["meta"]["status"] = "Codex quota"
    return state


def add_codex_subscription_tooltip(state):
    if state["subscription"].get("status") != "live":
        return state
    if "Codex:" in state.get("tooltip", ""):
        return state
    state["tooltip"] += (
        f"\nCodex: {state['subscription']['plan']}"
        f" · Session {state['subscription']['session']['value']}"
        f" · Weekly {state['subscription']['weekly']['value']}"
    )
    return state


def ai_usage_state_from_json(
    daily_json,
    ccusage_blocks_json,
    openusage_blocks_json,
    now_epoch=None,
    codex_subscription_json="",
    codex_reset_credits_json="",
):
    now = now_epoch or time.time()
    ccusage_state_value = ccusage_state_from_json(daily_json, ccusage_blocks_json, now_epoch=now)
    if ccusage_state_value == CCUSAGE_DEFAULT.copy() and not openusage_blocks_json and not codex_subscription_json:
        return deepcopy(AI_USAGE_DEFAULT)

    state = deepcopy(AI_USAGE_DEFAULT)
    state["updated"] = time.strftime("%H:%M", time.localtime(now))
    state["today"] = {
        **state["today"],
        **ccusage_state_value.get("today", {}),
    }
    state["text"] = f"󱃖 {state['today']['tokens']}"
    state["providers"] = provider_cards_from_agents(agents_from_daily_json(daily_json, now_epoch=now))
    if codex_subscription_json:
        subscription = codex_subscription_state_from_json(
            codex_subscription_json,
            codex_reset_credits_json,
            now_epoch=now,
        )
        apply_codex_subscription(state, subscription)

    daily_report = parse_json(daily_json, {})
    token_values = daily_token_values(daily_row_from_report(daily_report, now_epoch=now))
    raw_total = token_values["input"] + token_values["output"] + token_values["cache"]
    if raw_total <= 0:
        raw_total = token_values["total"]
    state["today"]["input_percent"] = percent_part(token_values["input"], raw_total)
    state["today"]["output_percent"] = percent_part(token_values["output"], raw_total)
    state["today"]["cache_percent"] = percent_part(token_values["cache"], raw_total)

    openusage_block = openusage_block_from_json(openusage_blocks_json, now_epoch=now)
    if openusage_block:
        state["block"] = openusage_block
        state["source"] = "openusage"
        state["meta"]["status"] = "OpenUsage"
    else:
        state["block"] = {
            **state["block"],
            **ccusage_state_value.get("block", {}),
            "burn_rate": "--",
            "source": "ccusage",
            "label": "Active block",
        }
        state["source"] = "ccusage" if ccusage_state_value != CCUSAGE_DEFAULT.copy() else "missing"
        state["meta"]["status"] = "ccusage fallback"

    cls = "active" if state["source"] != "missing" else "missing"
    today_cost = number_value({"cost": state["today"].get("cost", "--").lstrip("$")}, "cost")
    if today_cost >= 20 or state["block"].get("percent", 0) >= 85:
        cls = "warning"
    state["class"] = cls
    state["tooltip"] = (
        f"AI usage via {state['block']['source']}\n"
        f"Today: {state['today']['tokens']} / {state['today']['cost']}\n"
        f"Block: {state['block']['tokens']} used, {state['block']['remaining']} left\n"
        f"Projected: {state['block']['projected_cost']}"
    )
    return add_codex_subscription_tooltip(state)


_LAST_AI_USAGE = None


def ai_usage_state():
    global _LAST_AI_USAGE

    today = time.strftime("%F")
    daily = run_text(
        [
            "ccusage",
            "daily",
            "--json",
            "--offline",
            "--since",
            today,
            "--until",
            today,
        ],
        timeout=15.0,
    )
    openusage_blocks = run_text(["openusage", "blocks", "--json", "--offline"], timeout=8.0)
    ccusage_blocks = "" if openusage_blocks else run_text(["ccusage", "blocks", "--json", "--offline"], timeout=15.0)
    value = ai_usage_state_from_json(daily, ccusage_blocks, openusage_blocks)
    apply_codex_subscription(value, codex_subscription_state())
    add_codex_subscription_tooltip(value)
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


def network_state():
    status = run_text(["nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status"])
    wifi_device = connected_device(status, "wifi")
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

    ethernet_device = connected_device(status, "ethernet")
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

    return {
        "text": "⚠ Disconnected",
        "tooltip": "No connection",
        "class": "disconnected",
    }


def volume_state_from_text(text):
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


def volume_state():
    return volume_state_from_text(run_text(["wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"]))


def battery_state():
    for battery in Path("/sys/class/power_supply").glob("BAT*"):
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

        time_str = "N/A"
        now = rate = full_value = None
        for now_name, rate_name, full_name in [
            ("energy_now", "power_now", "energy_full"),
            ("charge_now", "current_now", "charge_full"),
        ]:
            try:
                now = int((battery / now_name).read_text().strip())
                rate = int((battery / rate_name).read_text().strip())
                full_value = int((battery / full_name).read_text().strip())
                break
            except Exception:
                now = rate = full_value = None

        minutes = None
        if now is not None and rate and rate > 0:
            if status == "Discharging":
                minutes = now * 60 // rate
            elif status == "Charging" and full_value:
                minutes = (full_value - now) * 60 // rate
        if minutes is not None:
            time_str = f"{minutes // 60}h {minutes % 60}m"

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
    devices = [line for line in devices_text.splitlines() if line.strip()]
    if not devices:
        tooltip = f"{controller_alias}\t{controller_address}\n\n0 connected"
        return {"text": "", "tooltip": tooltip, "class": powered}

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
    for line in devices:
        parts = line.split(maxsplit=2)
        if len(parts) < 2:
            continue
        address = parts[1]
        fallback = parts[2] if len(parts) >= 3 else address
        alias, battery = parse_device_info(info_by_address.get(address, ""), fallback)
        if battery:
            detail_lines.append(f"{alias}\t{address}\t{battery}")
        else:
            detail_lines.append(f"{alias}\t{address}")

    tooltip = (
        f"{controller_alias}\t{controller_address}\n\n"
        f"{len(devices)} connected\n\n"
        + "\n".join(detail_lines)
    )
    return {"text": text, "tooltip": tooltip, "class": "connected"}


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
