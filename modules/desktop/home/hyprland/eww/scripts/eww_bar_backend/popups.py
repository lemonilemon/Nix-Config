import subprocess
import sys

# The full set of popup windows the helper manages. Closing a window that is not
# open is a harmless no-op in eww, so `close` can name all of them at once.
POPUP_WINDOWS = [
    "volume_popup",
    "bluetooth_popup",
    "network_popup",
    "battery_popup",
    "ai_usage_popup",
    "display_mode_popup",
]
BACKDROP_WINDOW = "popup_backdrop"

POPUP_USAGE = "usage: eww-popup toggle <window> <screen> | close"


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
    try:
        return subprocess.check_output(
            ["eww", "active-windows"], text=True, stderr=subprocess.DEVNULL
        )
    except Exception:
        return ""


def _run_eww(args):
    subprocess.run(
        ["eww", *args],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )


def run_popup(argv, active_windows_fn=_active_windows_text, eww_fn=_run_eww):
    if not argv or argv[0] in ("-h", "--help", "help"):
        print(POPUP_USAGE, file=sys.stderr)
        return 1
    command, args = argv[0], argv[1:]
    open_ids = parse_open_windows(active_windows_fn()) if command == "toggle" else set()
    try:
        calls = popup_eww_calls(command, args, open_ids)
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    for call in calls:
        eww_fn(call)
    return 0
