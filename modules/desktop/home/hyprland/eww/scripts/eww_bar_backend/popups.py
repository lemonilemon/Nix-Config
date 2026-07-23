import subprocess
import sys

from .common import parse_json, run_text

# The full set of popup windows the helper manages. `eww close` is fire-and-forget
# here (via _run_eww with check=False): naming a window that is closed or not yet
# defined just prints a warning and exits non-zero, which we ignore — so `close`
# can safely name every popup plus the backdrop at once.
POPUP_WINDOWS = [
    "volume_popup",
    "bluetooth_popup",
    "network_popup",
    "battery_popup",
    "ai_usage_popup",
    "display_mode_popup",
    "notif_center_popup",
    "wallpaper_picker_popup",
    "settings_popup",
]
BACKDROP_WINDOW = "popup_backdrop"

POPUP_USAGE = "usage: eww-popup toggle <window> [screen] | close"


def parse_open_windows(active_windows_text):
    # `eww active-windows` prints one "<id>: <window-name>" per line. A popup
    # opened without an explicit --id has id == name, so the id (left of ":")
    # identifies whether that popup is currently open.
    open_ids = set()
    for line in active_windows_text.splitlines():
        line = line.strip()
        if not line or ":" not in line:
            continue
        open_ids.add(line.split(":", 1)[0].strip())
    return open_ids


def focused_monitor_from_json(monitors_json):
    data = parse_json(monitors_json, [])
    if isinstance(data, list):
        for monitor in data:
            if isinstance(monitor, dict) and monitor.get("focused") is True:
                name = monitor.get("name")
                if isinstance(name, str) and name:
                    return name
    return "0"


def _focused_monitor():
    return focused_monitor_from_json(run_text(["hyprctl", "monitors", "-j"]))


def _close_call():
    return ["close", *POPUP_WINDOWS, BACKDROP_WINDOW]


def popup_eww_calls(command, args, open_ids):
    if command == "close":
        return [_close_call()]
    if command == "toggle":
        if len(args) != 2:
            raise ValueError(POPUP_USAGE)
        name, screen = args
        if name not in POPUP_WINDOWS:
            raise ValueError(f"unknown popup window: {name}")
        calls = [_close_call()]
        if name not in open_ids:
            # Was closed: open the backdrop first so the popup stacks above it,
            # then the popup, both on the clicked screen. (If it was open, the
            # close above already dismissed it — a toggle-off.)
            calls.append(["open", BACKDROP_WINDOW, "--screen", screen])
            calls.append(["open", name, "--screen", screen])
        return calls
    raise ValueError(POPUP_USAGE)


def _active_windows_text():
    # Reuse the package's shared runner so this inherits the same 2s timeout and
    # empty-on-failure behavior as every other collector, rather than blocking
    # forever if the eww daemon is running but wedged.
    return run_text(["eww", "active-windows"], timeout=2.0)


def _run_eww(args):
    subprocess.run(
        ["eww", *args],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )


def run_popup(argv, active_windows_fn=_active_windows_text, eww_fn=_run_eww, monitor_fn=_focused_monitor):
    if not argv or argv[0] in ("-h", "--help", "help"):
        print(POPUP_USAGE, file=sys.stderr)
        return 1
    command, args = argv[0], argv[1:]
    if command == "toggle" and len(args) == 1:
        args = [args[0], monitor_fn()]
    open_ids = parse_open_windows(active_windows_fn()) if command == "toggle" else set()
    try:
        calls = popup_eww_calls(command, args, open_ids)
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    for call in calls:
        eww_fn(call)
    return 0
