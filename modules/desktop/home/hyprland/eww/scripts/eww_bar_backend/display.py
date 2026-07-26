import subprocess

from .common import parse_json, run_text
from .paths import display_mode_path
from .inhibitors import (
    idle_inhibited_state,
    lid_inhibited_state,
    set_idle_inhibited,
    set_lid_inhibited,
)


DISPLAY_MODES = {"normal", "external", "headless"}
INTERNAL_MONITOR_PREFIXES = ("eDP-", "LVDS-")

MODE_STATUS = {
    "normal": "Normal desktop mode",
    "external": "External display mode",
    "headless": "Headless server mode",
}


def read_display_mode():
    try:
        mode = display_mode_path().read_text().strip()
    except Exception:
        return "normal"
    return mode if mode in DISPLAY_MODES else "normal"


def write_display_mode(mode):
    path = display_mode_path()
    if mode == "normal":
        try:
            path.unlink()
        except FileNotFoundError:
            pass
        except Exception:
            pass
        return
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(mode)


def display_state(status=""):
    mode = read_display_mode()
    return {
        "mode": mode,
        "lid_inhibited": lid_inhibited_state(),
        "status": status or MODE_STATUS[mode],
        "class": mode,
    }


def monitor_state():
    return parse_json(run_text(["hyprctl", "monitors", "-j"]), [])


def is_internal_monitor(name):
    return name.startswith(INTERNAL_MONITOR_PREFIXES)


def split_monitors(monitors):
    enabled = [monitor for monitor in monitors if not monitor.get("disabled", False)]
    internal = [monitor for monitor in enabled if is_internal_monitor(monitor.get("name", ""))]
    external = [monitor for monitor in enabled if not is_internal_monitor(monitor.get("name", ""))]
    return internal, external


def run_hyprctl(*args):
    result = subprocess.run(
        ["hyprctl", *args],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    if result.returncode != 0:
        raise RuntimeError(f"hyprctl {' '.join(args)} failed")


def set_display_mode(action):
    if action == "status":
        return display_state()

    if action == "toggle":
        # Toggle headless on/off. This is the wake path when the screen is off:
        # the popup "Turn screens on" button is unreachable in headless mode
        # (it renders on the display that was powered off), so a keybind bound
        # to this action lets the user press once to sleep and again to cancel.
        if read_display_mode() == "headless":
            return set_display_mode("normal")
        return set_display_mode("headless")

    if action == "restore":
        run_hyprctl("dispatch", "dpms", "on")
        return display_state("Screens turned on")

    if action == "normal":
        run_hyprctl("dispatch", "dpms", "on")
        run_hyprctl("reload")
        set_lid_inhibited(False)
        set_idle_inhibited(False)
        write_display_mode("normal")
        return display_state("Exited display mode")

    if action == "external":
        internal, external = split_monitors(monitor_state())
        if not external:
            raise ValueError("external mode requires an active external monitor")
        set_idle_inhibited(True)
        set_lid_inhibited(True)
        # Guarantee the external monitor is powered on: an "enabled" monitor can
        # still be DPMS-off (e.g. arriving from headless or an idle blank), in
        # which case disabling the internal panel would leave a black screen.
        run_hyprctl("dispatch", "dpms", "on")
        for monitor in internal:
            name = monitor.get("name", "")
            if name:
                run_hyprctl("keyword", "monitor", f"{name},disable")
        write_display_mode("external")
        return display_state("External monitors active")

    if action == "headless":
        set_idle_inhibited(True)
        set_lid_inhibited(True)
        run_hyprctl("dispatch", "dpms", "off")
        write_display_mode("headless")
        return display_state("Headless server mode")

    raise ValueError("display action must be normal, external, headless, restore, toggle, or status")
