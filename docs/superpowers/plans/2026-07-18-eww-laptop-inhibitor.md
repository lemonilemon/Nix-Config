# Eww Laptop Inhibitor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an Eww-native stay-awake/display-mode control that pauses Hypridle, supports laptop clamshell/headless presets, and keeps desktop behavior unchanged.

**Architecture:** Replace PID-file inhibitor tracking with systemd user services and expose backend state through the existing `eww-barctl` Unix socket. Add a focused display-mode backend module for Hyprland monitor actions, then render a laptop-only Eww popup modeled after the existing AI usage panel.

**Tech Stack:** NixOS, Home Manager, Hyprland, Eww, systemd user services, Python standard library `unittest`, shell verification through `just`.

---

## File Structure

- Create `tests/eww_bar_backend/test_inhibitors_display.py`: standard-library unit tests for service-backed inhibitors and display-mode actions.
- Create `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/inhibitors.py`: systemd user-service helpers and idle/lid inhibitor state.
- Create `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/display.py`: display-mode state file, monitor detection, and Hyprland commands.
- Modify `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/common.py`: add `display_mode_path()`.
- Modify `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/control.py`: route `idle` through `inhibitors.py`, add `display` commands.
- Modify `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/state.py`: add `display` state defaults.
- Modify `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/app.py`: initialize display state.
- Modify `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/watchers.py`: refresh display and inhibitor state together.
- Modify `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/__init__.py`: export the new backend helpers.
- Delete `modules/desktop/home/hyprland/eww/scripts/idle-toggle`: remove the obsolete shell implementation.
- Modify `modules/desktop/home/hyprland/eww/default.nix`: generate `eww.yuck` with a laptop-controls flag and define the inhibitor services.
- Modify `modules/desktop/options.nix` and `modules/desktop/home/options.nix`: declare `home.desktop.hyprland.eww.laptopControls.enable`.
- Modify `profiles/laptop/config.nix`: enable laptop controls for the laptop profile.
- Modify `modules/desktop/home/hyprland/default.nix`: add laptop-only display restore keybind.
- Modify `modules/desktop/home/hyprland/eww/eww.yuck`: add the right-click popup UI and updated button behavior.
- Modify `modules/desktop/home/hyprland/eww/eww.scss`: style the display-mode popup.
- Modify `modules/desktop/README.md`: document the Eww laptop controls flag and behavior.

---

### Task 1: Add Backend Tests

**Files:**
- Create: `tests/eww_bar_backend/test_inhibitors_display.py`

- [ ] **Step 1: Create the failing unit tests**

Create `tests/eww_bar_backend/test_inhibitors_display.py` with this content:

```python
import os
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch


REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import display, inhibitors  # noqa: E402


class InhibitorTests(unittest.TestCase):
    def test_set_idle_inhibited_starts_user_service(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        with patch.object(inhibitors.subprocess, "run", side_effect=fake_run):
            with patch.object(inhibitors, "service_active", return_value=True):
                self.assertEqual(inhibitors.set_idle_inhibited(True), "true")

        self.assertEqual(
            calls,
            [["systemctl", "--user", "start", "eww-hypridle-inhibit.service"]],
        )

    def test_set_idle_inhibited_stops_user_service(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        with patch.object(inhibitors.subprocess, "run", side_effect=fake_run):
            with patch.object(inhibitors, "service_active", return_value=False):
                self.assertEqual(inhibitors.set_idle_inhibited(False), "false")

        self.assertEqual(
            calls,
            [["systemctl", "--user", "stop", "eww-hypridle-inhibit.service"]],
        )


class DisplayModeTests(unittest.TestCase):
    def test_external_mode_requires_an_external_monitor(self):
        commands = []
        monitor_data = [{"name": "eDP-1", "disabled": False}]

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                with patch.object(display, "monitor_state", return_value=monitor_data):
                    with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                        with self.assertRaisesRegex(ValueError, "external monitor"):
                            display.set_display_mode("external")

        self.assertEqual(commands, [])

    def test_external_mode_disables_internal_panel_and_keeps_inhibitors(self):
        commands = []
        monitor_data = [
            {"name": "eDP-1", "disabled": False},
            {"name": "HDMI-A-1", "disabled": False},
        ]

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                with patch.object(display, "monitor_state", return_value=monitor_data):
                    with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                        with patch.object(display, "set_idle_inhibited", return_value="true"):
                            with patch.object(display, "set_lid_inhibited", return_value="true"):
                                with patch.object(display, "lid_inhibited_state", return_value="true"):
                                    result = display.set_display_mode("external")

        self.assertEqual(result["mode"], "external")
        self.assertEqual(result["lid_inhibited"], "true")
        self.assertIn(("keyword", "monitor", "eDP-1,disable"), commands)

    def test_headless_mode_turns_dpms_off(self):
        commands = []

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                    with patch.object(display, "set_idle_inhibited", return_value="true"):
                        with patch.object(display, "set_lid_inhibited", return_value="true"):
                            with patch.object(display, "lid_inhibited_state", return_value="true"):
                                result = display.set_display_mode("headless")

        self.assertEqual(result["mode"], "headless")
        self.assertEqual(result["lid_inhibited"], "true")
        self.assertEqual(commands, [("dispatch", "dpms", "off")])

    def test_restore_turns_screens_on_without_clearing_mode(self):
        commands = []

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                display.write_display_mode("headless")
                with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                    with patch.object(display, "lid_inhibited_state", return_value="true"):
                        result = display.set_display_mode("restore")

        self.assertEqual(result["mode"], "headless")
        self.assertEqual(result["lid_inhibited"], "true")
        self.assertEqual(commands, [("dispatch", "dpms", "on")])

    def test_normal_mode_restores_and_clears_inhibitors(self):
        commands = []

        with TemporaryDirectory() as runtime_dir:
            with patch.dict(os.environ, {"XDG_RUNTIME_DIR": runtime_dir}):
                display.write_display_mode("external")
                with patch.object(display, "run_hyprctl", side_effect=lambda *args: commands.append(args)):
                    with patch.object(display, "set_idle_inhibited", return_value="false"):
                        with patch.object(display, "set_lid_inhibited", return_value="false"):
                            with patch.object(display, "lid_inhibited_state", return_value="false"):
                                result = display.set_display_mode("normal")

        self.assertEqual(result["mode"], "normal")
        self.assertEqual(result["lid_inhibited"], "false")
        self.assertEqual(commands, [("dispatch", "dpms", "on"), ("reload",)])


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run the tests and confirm they fail for missing modules**

Run:

```bash
python3 -m unittest discover -s tests -p 'test_*.py'
```

Expected: FAIL with an import error containing `cannot import name 'display'` or `cannot import name 'inhibitors'`.

- [ ] **Step 3: Commit the failing tests**

```bash
git add tests/eww_bar_backend/test_inhibitors_display.py
git commit -m "test(eww): cover inhibitor and display mode backend"
```

---

### Task 2: Implement Backend Services And Display Commands

**Files:**
- Create: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/inhibitors.py`
- Create: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/display.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/common.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/control.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/state.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/app.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/watchers.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/__init__.py`
- Delete: `modules/desktop/home/hyprland/eww/scripts/idle-toggle`

- [ ] **Step 1: Add systemd-backed inhibitor helpers**

Create `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/inhibitors.py`:

```python
import subprocess


HYPRIDLE_INHIBIT_SERVICE = "eww-hypridle-inhibit.service"
LID_INHIBIT_SERVICE = "eww-lid-inhibit.service"


def bool_state(value):
    return "true" if value else "false"


def service_active(service):
    result = subprocess.run(
        ["systemctl", "--user", "is-active", "--quiet", service],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    return result.returncode == 0


def set_service_active(service, enabled):
    action = "start" if enabled else "stop"
    result = subprocess.run(
        ["systemctl", "--user", action, service],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    if result.returncode != 0:
        raise RuntimeError(f"systemctl --user {action} {service} failed")
    return service_active(service)


def idle_inhibited_state():
    return bool_state(service_active(HYPRIDLE_INHIBIT_SERVICE))


def lid_inhibited_state():
    return bool_state(service_active(LID_INHIBIT_SERVICE))


def set_idle_inhibited(enabled):
    return bool_state(set_service_active(HYPRIDLE_INHIBIT_SERVICE, enabled))


def set_lid_inhibited(enabled):
    return bool_state(set_service_active(LID_INHIBIT_SERVICE, enabled))


def toggle_idle_inhibited():
    return set_idle_inhibited(not service_active(HYPRIDLE_INHIBIT_SERVICE))
```

- [ ] **Step 2: Add display mode path helper**

In `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/common.py`, add this after `idle_pidfile_path()`:

```python
def display_mode_path():
    return runtime_file("eww-display-mode")
```

- [ ] **Step 3: Add display-mode backend**

Create `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/display.py`:

```python
import subprocess

from .common import display_mode_path, parse_json, run_text
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

    raise ValueError("display action must be normal, external, headless, restore, or status")
```

- [ ] **Step 4: Wire display and inhibitor state into control commands**

In `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/control.py`:

Replace these imports:

```python
import signal
from pathlib import Path

from .collectors import idle_inhibited_state, media_state, refresh_ai_usage, volume_state
from .common import backend_pidfile_path, control_socket_path, idle_pidfile_path, parse_json
```

with:

```python
from .collectors import media_state, refresh_ai_usage, volume_state
from .common import backend_pidfile_path, control_socket_path, parse_json
from .display import display_state, set_display_mode
from .inhibitors import idle_inhibited_state, set_idle_inhibited, toggle_idle_inhibited
```

Replace `CONTROL_USAGE` with:

```python
CONTROL_USAGE = (
    "usage: eww-barctl ping | volume up|down | media play-pause|next|previous | "
    "idle toggle|on|off|status | display normal|external|headless|restore|status | ai refresh"
)
```

Delete `live_pid_from_file()`, `set_idle_inhibited()`, and `toggle_idle_inhibited()` from `control.py`.

In `handle_control_command()`, replace the `idle` block with:

```python
    if command == "idle":
        action = payload.get("action", "toggle")
        if action == "toggle":
            value = toggle_idle_inhibited()
        elif action == "on":
            value = set_idle_inhibited(True)
        elif action == "off":
            value = set_idle_inhibited(False)
        elif action == "status":
            value = idle_inhibited_state()
        else:
            raise ValueError("idle action must be toggle, on, off, or status")
        state.update(idle_inhibited=value, display=display_state())
        return {"ok": True, "command": "idle", "idle_inhibited": value}
```

Add this block immediately after the `idle` block:

```python
    if command == "display":
        action = payload.get("action", "status")
        value = set_display_mode(action)
        state.update(idle_inhibited=idle_inhibited_state(), display=value)
        return {"ok": True, "command": "display", "action": action, "display": value}
```

In `control_payload_from_args()`, add this before the `ai` block:

```python
    if args[0] == "display":
        action = args[1] if len(args) > 1 else "status"
        return {"command": "display", "action": action}
```

- [ ] **Step 5: Add display defaults to state and app startup**

In `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/state.py`, add this key after `"idle_inhibited": "false",`:

```python
            "display": {
                "mode": "normal",
                "lid_inhibited": "false",
                "status": "Normal desktop mode",
                "class": "normal",
            },
```

In `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/app.py`, add:

```python
from .display import display_state
from .inhibitors import idle_inhibited_state
```

Remove `idle_inhibited_state` from the `.collectors` import list.

In the initial `state.update(...)`, add:

```python
        display=display_state(),
```

- [ ] **Step 6: Refresh display state with the existing inhibitor watcher**

In `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/watchers.py`, replace:

```python
from .collectors import (
    active_window_state,
    bluetooth_state,
    idle_inhibited_state,
    network_state,
    submap_from_event,
    volume_state,
    workspace_state,
)
```

with:

```python
from .collectors import (
    active_window_state,
    bluetooth_state,
    network_state,
    submap_from_event,
    volume_state,
    workspace_state,
)
from .display import display_state
from .inhibitors import idle_inhibited_state
```

Replace `watch_idle_inhibitor()` with:

```python
def watch_idle_inhibitor(state, refresh_event):
    state.update(idle_inhibited=idle_inhibited_state(), display=display_state())
    while True:
        refresh_event.wait(timeout=30)
        refresh_event.clear()
        state.update(idle_inhibited=idle_inhibited_state(), display=display_state())
```

- [ ] **Step 7: Update package exports**

In `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/__init__.py`:

Remove `idle_inhibited_state` from the `.collectors` import list.

Add:

```python
from .display import (
    display_state,
    is_internal_monitor,
    monitor_state,
    read_display_mode,
    run_hyprctl,
    set_display_mode,
    split_monitors,
    write_display_mode,
)
from .inhibitors import (
    HYPRIDLE_INHIBIT_SERVICE,
    LID_INHIBIT_SERVICE,
    idle_inhibited_state,
    lid_inhibited_state,
    service_active,
    set_idle_inhibited,
    set_lid_inhibited,
    toggle_idle_inhibited,
)
```

Remove these names from the `.control` import list:

```python
    live_pid_from_file,
    set_idle_inhibited,
    toggle_idle_inhibited,
```

Add `display_mode_path` to the `.common` import list.

- [ ] **Step 8: Remove obsolete shell toggle**

Run:

```bash
git rm modules/desktop/home/hyprland/eww/scripts/idle-toggle
```

- [ ] **Step 9: Run backend tests**

Run:

```bash
python3 -m unittest discover -s tests -p 'test_*.py'
```

Expected: `Ran 7 tests` and `OK`.

- [ ] **Step 10: Compile backend Python files**

Run:

```bash
python3 -m py_compile \
  modules/desktop/home/hyprland/eww/scripts/backend \
  modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/*.py
```

Expected: no output and exit code `0`.

- [ ] **Step 11: Commit backend implementation**

```bash
git add \
  modules/desktop/home/hyprland/eww/scripts/backend \
  modules/desktop/home/hyprland/eww/scripts/eww_bar_backend \
  modules/desktop/home/hyprland/eww/scripts/idle-toggle
git commit -m "feat(eww): manage idle and display modes in backend"
```

---

### Task 3: Add Nix Options, Services, And Laptop Gate

**Files:**
- Modify: `modules/desktop/options.nix`
- Modify: `modules/desktop/home/options.nix`
- Modify: `modules/desktop/home/hyprland/eww/default.nix`
- Modify: `modules/desktop/home/hyprland/default.nix`
- Modify: `profiles/laptop/config.nix`

- [ ] **Step 1: Add the top-level option mirror**

In `modules/desktop/options.nix`, add this after `hyprland.eww.enable`:

```nix
      hyprland.eww.laptopControls.enable = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = "Enable laptop-only Eww controls for clamshell and headless display modes";
      };
```

- [ ] **Step 2: Add the Home Manager option mirror**

In `modules/desktop/home/options.nix`, add this after `hyprland.eww.enable`:

```nix
      hyprland.eww.laptopControls.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.eww.laptopControls.enable";
        default = false;
        description = "Enable laptop-only Eww controls for clamshell and headless display modes";
      };
```

- [ ] **Step 3: Enable laptop controls in the laptop profile**

In `profiles/laptop/config.nix`, change the body to:

```nix
{
  ...
}:
{
  home.desktop.hyprland.eww.enable = true;
  home.desktop.hyprland.eww.laptopControls.enable = true;

  nixos.desktop.gnome.enable = false;
}
```

- [ ] **Step 4: Generate `eww.yuck` with the laptop flag**

In `modules/desktop/home/hyprland/eww/default.nix`, add this to the `let` block after `runtimePath`:

```nix
  ewwYuck = pkgs.substituteAll {
    src = ./eww.yuck;
    laptopControls = if cfg.laptopControls.enable then "true" else "false";
  };
```

Then change:

```nix
    xdg.configFile."eww/eww.yuck".source = ./eww.yuck;
```

to:

```nix
    xdg.configFile."eww/eww.yuck".source = ewwYuck;
```

- [ ] **Step 5: Define the systemd user inhibitor services**

In `modules/desktop/home/hyprland/eww/default.nix`, replace the single `systemd.user.services.eww-bar = { ... };` assignment with:

```nix
    systemd.user.services = {
      eww-bar = {
        Unit = {
          Description = "Eww Hyprland bar";
          After = [ "hyprland-session.target" ];
          PartOf = [ "hyprland-session.target" ];
        };

        Service = {
          Environment = [ "PATH=${runtimePath}" ];
          ExecStart = "${pkgs.eww}/bin/eww --force-wayland daemon --no-daemonize";
          ExecStartPost = openBar;
          MemoryAccounting = true;
          Restart = "on-failure";
          RestartSec = "1s";
        };

        Install.WantedBy = [ "hyprland-session.target" ];
      };

      eww-hypridle-inhibit = {
        Unit.Description = "Eww Hypridle inhibitor";
        Service = {
          Type = "simple";
          Environment = [ "PATH=${runtimePath}" ];
          ExecStart = "${pkgs.systemd}/bin/systemd-inhibit --what=idle --who=eww-bar --why=Hypridle paused from Eww ${pkgs.coreutils}/bin/sleep infinity";
        };
      };
    }
    // lib.optionalAttrs cfg.laptopControls.enable {
      eww-lid-inhibit = {
        Unit.Description = "Eww laptop lid inhibitor";
        Service = {
          Type = "simple";
          Environment = [ "PATH=${runtimePath}" ];
          ExecStart = "${pkgs.systemd}/bin/systemd-inhibit --what=handle-lid-switch --who=eww-bar --why=Eww laptop display mode keeps lid close ignored ${pkgs.coreutils}/bin/sleep infinity";
        };
      };
    };
```

- [ ] **Step 6: Add the laptop-only restore keybind**

In `modules/desktop/home/hyprland/default.nix`, extend the `bind` list by adding this after the screenshot bindings:

```nix
          ++ lib.optionals (
            config.home.desktop.hyprland.eww.enable
            && config.home.desktop.hyprland.eww.laptopControls.enable
          ) [
            "${MOD1}+SHIFT, o, exec, eww-barctl display restore"
          ]
```

Keep the existing workspace-generation append after this new append. The resulting shape should be:

```nix
          bind =
            [
              # existing static bindings
            ]
            ++ lib.optionals (
              config.home.desktop.hyprland.eww.enable
              && config.home.desktop.hyprland.eww.laptopControls.enable
            ) [
              "${MOD1}+SHIFT, o, exec, eww-barctl display restore"
            ]
            ++ (
              # existing workspace binding generation
            );
```

- [ ] **Step 7: Format and evaluate laptop and desktop configs**

Run:

```bash
just fmt
NIXHOST=laptop just test
NIXHOST=desktop just test
```

Expected: all three commands exit `0`.

- [ ] **Step 8: Commit Nix service and option wiring**

```bash
git add \
  modules/desktop/options.nix \
  modules/desktop/home/options.nix \
  modules/desktop/home/hyprland/eww/default.nix \
  modules/desktop/home/hyprland/default.nix \
  profiles/laptop/config.nix
git commit -m "feat(eww): add laptop display mode services"
```

---

### Task 4: Build The Eww Popup Frontend

**Files:**
- Modify: `modules/desktop/home/hyprland/eww/eww.yuck`
- Modify: `modules/desktop/home/hyprland/eww/eww.scss`

- [ ] **Step 1: Add a generated laptop-controls variable**

In `modules/desktop/home/hyprland/eww/eww.yuck`, add this after the `deflisten`:

```lisp
(defvar laptop_controls "@laptopControls@")
```

In the initial `bar_state` JSON string, add this field before the final closing brace:

```json
,"display":{"mode":"normal","lid_inhibited":"false","status":"Normal desktop mode","class":"normal"}
```

- [ ] **Step 2: Update the inhibitor button**

Replace the existing idle inhibitor button:

```lisp
          (button :class "idle-inhibitor ${bar_state.idle_inhibited == 'true' ? 'active' : ''}"
            :onclick "eww-barctl --quiet idle toggle"
            (label :text {bar_state.idle_inhibited == "true" ? "" : ""}))
```

with:

```lisp
          (button :class "idle-inhibitor ${bar_state.idle_inhibited == 'true' ? 'active' : (bar_state.display.mode != 'normal' ? 'active' : '')} ${bar_state.display.class}"
            :onclick "eww-barctl --quiet idle toggle"
            :onrightclick {laptop_controls == "true" ? "eww open --toggle display_mode_popup --screen ${output}" : "true"}
            :tooltip {bar_state.display.mode == "normal" ? (bar_state.idle_inhibited == "true" ? "Hypridle paused" : "Hypridle active") : bar_state.display.status}
            (label :text {bar_state.display.mode == "headless" ? "󰒲" : (bar_state.display.mode == "external" ? "󰍹" : (bar_state.idle_inhibited == "true" ? "" : ""))}))
```

- [ ] **Step 3: Add the display-mode popup window and widget**

Add this after the `workspaces` widgets and before `ai_usage_popup`:

```lisp
(defwindow display_mode_popup
  :monitor 0
  :geometry (geometry
    :x "12px"
    :y "50px"
    :width "300px"
    :height "252px"
    :anchor "top right")
  :stacking "fg"
  :exclusive false
  :focusable false
  :namespace "eww-display-mode"
  (display_mode_panel))

(defwidget display_mode_panel []
  (box :class "display-mode-popup" :orientation "v" :space-evenly false
    (box :class "display-mode-header" :orientation "h" :space-evenly false
      (box :class "display-mode-mark ${bar_state.display.class}"
        (label :text {bar_state.display.mode == "headless" ? "󰒲" : (bar_state.display.mode == "external" ? "󰍹" : "")}))
      (box :class "display-mode-title-stack" :orientation "v" :space-evenly false
        (label :class "display-mode-title" :halign "start" :text {bar_state.display.mode == "headless" ? "Headless server" : (bar_state.display.mode == "external" ? "External only" : "Stay awake")})
        (label :class "display-mode-eyebrow" :halign "start" :text {bar_state.display.status})))
    (box :class "display-mode-status" :orientation "h" :space-evenly false
      (label :class "display-mode-chip ${bar_state.idle_inhibited == 'true' ? 'active' : ''}" :text {bar_state.idle_inhibited == "true" ? "Hypridle paused" : "Hypridle active"})
      (label :class "display-mode-chip ${bar_state.display.lid_inhibited == 'true' ? 'active' : ''}" :text {bar_state.display.lid_inhibited == "true" ? "Lid ignored" : "Lid normal"}))
    (box :class "display-mode-actions" :orientation "v" :space-evenly false
      (button :class "display-mode-action ${bar_state.idle_inhibited == 'true' ? 'active' : ''}"
        :onclick "eww-barctl --quiet idle toggle"
        (box :orientation "h" :space-evenly false
          (label :class "display-mode-action-icon" :text "")
          (label :class "display-mode-action-label" :halign "start" :text "Pause Hypridle")))
      (button :class "display-mode-action ${bar_state.display.mode == 'external' ? 'active' : ''}"
        :onclick "eww-barctl --quiet display external"
        (box :orientation "h" :space-evenly false
          (label :class "display-mode-action-icon" :text "󰍹")
          (label :class "display-mode-action-label" :halign "start" :text "External only")))
      (button :class "display-mode-action ${bar_state.display.mode == 'headless' ? 'active' : ''}"
        :onclick "eww-barctl --quiet display headless"
        (box :orientation "h" :space-evenly false
          (label :class "display-mode-action-icon" :text "󰒲")
          (label :class "display-mode-action-label" :halign "start" :text "Headless server"))))
    (box :class "display-mode-footer" :orientation "h" :space-evenly false
      (button :class "display-mode-footer-action"
        :onclick "eww-barctl --quiet display restore"
        (label :text "Turn screens on"))
      (button :class "display-mode-footer-action danger"
        :onclick "eww-barctl --quiet display normal && eww close display_mode_popup"
        (label :text "Exit mode")))))
```

- [ ] **Step 4: Add popup styling**

In `modules/desktop/home/hyprland/eww/eww.scss`, add this before `.ai-usage-popup`:

```scss
.display-mode-popup {
  background: rgba(17, 17, 27, 0.985);
  border: 1px solid rgba(205, 214, 244, 0.14);
  border-radius: 12px;
  box-shadow: 0 20px 48px rgba(7, 8, 17, 0.58);
  color: $text;
}

.display-mode-header {
  border-bottom: 1px solid rgba(205, 214, 244, 0.07);
  padding: 14px 15px 13px;
}

.display-mode-mark {
  background: rgba(249, 226, 175, 0.11);
  border: 1px solid rgba(249, 226, 175, 0.18);
  border-radius: 8px;
  color: $yellow;
  font-size: 15px;
  margin-right: 9px;
  min-height: 34px;
  min-width: 34px;
}

.display-mode-mark.external {
  background: rgba(137, 180, 250, 0.11);
  border-color: rgba(137, 180, 250, 0.18);
  color: $blue;
}

.display-mode-mark.headless {
  background: rgba(148, 226, 213, 0.11);
  border-color: rgba(148, 226, 213, 0.18);
  color: $teal;
}

.display-mode-title {
  color: $text;
  font-size: 16px;
  font-weight: 800;
}

.display-mode-eyebrow {
  color: $overlay1;
  font-size: 10px;
  margin-top: 1px;
}

.display-mode-status {
  margin: 10px 14px 0;
}

.display-mode-chip {
  background: rgba(49, 50, 68, 0.54);
  border-radius: 6px;
  color: $overlay1;
  font-size: 10px;
  margin-right: 6px;
  padding: 5px 8px;
}

.display-mode-chip.active {
  background: rgba(166, 227, 161, 0.10);
  color: $green;
}

.display-mode-actions {
  margin: 10px 14px 0;
}

.display-mode-action {
  border-radius: 7px;
  color: $subtext0;
  margin-bottom: 5px;
  min-height: 31px;
  padding: 0 9px;
}

.display-mode-action:hover,
.display-mode-footer-action:hover {
  background: rgba(49, 50, 68, 0.78);
  color: $text;
}

.display-mode-action.active {
  background: rgba(249, 226, 175, 0.10);
  color: $yellow;
}

.display-mode-action-icon {
  margin-right: 9px;
  min-width: 18px;
}

.display-mode-action-label {
  font-weight: 700;
}

.display-mode-footer {
  border-top: 1px solid rgba(205, 214, 244, 0.07);
  margin-top: 9px;
  padding: 10px 14px 12px;
}

.display-mode-footer-action {
  border-radius: 7px;
  color: $sapphire;
  font-size: 11px;
  font-weight: 800;
  margin-right: 7px;
  min-height: 28px;
  padding: 0 10px;
}

.display-mode-footer-action.danger {
  color: $red;
}

.idle-inhibitor.external {
  color: $blue;
}

.idle-inhibitor.headless {
  color: $teal;
}
```

- [ ] **Step 5: Evaluate generated Eww config through Nix**

Run:

```bash
NIXHOST=laptop just test
NIXHOST=desktop just test
```

Expected: both commands exit `0`.

- [ ] **Step 6: Commit Eww frontend**

```bash
git add modules/desktop/home/hyprland/eww/eww.yuck modules/desktop/home/hyprland/eww/eww.scss
git commit -m "feat(eww): add laptop display mode popup"
```

---

### Task 5: Document And Verify End To End

**Files:**
- Modify: `modules/desktop/README.md`

- [ ] **Step 1: Document the new Eww and laptop controls flags**

In `modules/desktop/README.md`, in the Home Manager options block, extend the example to:

```nix
home.desktop.enable                                  # Enable desktop home configuration
home.desktop.hyprland.enable                        # Enable Hyprland user config
home.desktop.hyprland.eww.enable                    # Enable Eww bar for Hyprland
home.desktop.hyprland.eww.laptopControls.enable     # Enable laptop clamshell/headless controls
home.desktop.hyprland.waybar.enable                 # Enable Waybar when Eww is disabled
```

Add this Eww section after the Waybar section:

```markdown
**Eww** (`eww/`):
- Top status bar replacement for Waybar
- Workspace, media, system, AI usage, and session controls
- Hypridle pause backed by a systemd user inhibitor
- Optional laptop controls for external-only and headless/server display modes
```

- [ ] **Step 2: Run all local verification**

Run:

```bash
python3 -m unittest discover -s tests -p 'test_*.py'
python3 -m py_compile \
  modules/desktop/home/hyprland/eww/scripts/backend \
  modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/*.py
just fmt
NIXHOST=laptop just test
NIXHOST=desktop just test
```

Expected:

```text
OK
```

from the unit tests, no output from `py_compile`, and exit code `0` from `just fmt`, `NIXHOST=laptop just test`, and `NIXHOST=desktop just test`.

- [ ] **Step 3: Run laptop dry build**

Run:

```bash
NIXHOST=laptop just dry-build
```

Expected: `nixos-rebuild dry-build` completes without evaluation or build errors. If the command requests sudo credentials, let the user enter them.

- [ ] **Step 4: Commit docs and verification cleanup**

```bash
git add modules/desktop/README.md
git commit -m "docs(desktop): document eww laptop controls"
```

---

## Manual Rebuild Checks

After the user rebuilds and switches the laptop profile:

- Left-click the inhibitor button. `systemctl --user is-active eww-hypridle-inhibit.service` should switch between `active` and `inactive`.
- Right-click the inhibitor button. The display-mode popup should open.
- Select `External only` with no external monitor connected. The mode should refuse to activate and keep the current visible display.
- Connect an external monitor and select `External only`. The internal panel should disable, the external monitor should remain active, and the lid can close without suspending.
- Select `Headless server`. Displays should turn off, Hypridle should stay paused, and lid-close handling should stay inhibited.
- Press `SUPER+SHIFT+O` or run `eww-barctl display restore`. Screens should turn on while mode state remains active.
- Select `Exit mode`. Screens should turn on, Hyprland should reload monitor config, `eww-hypridle-inhibit.service` should stop, and `eww-lid-inhibit.service` should stop.
- On desktop, right-clicking the inhibitor button should not expose laptop display controls.

---

## Plan Self-Review

- Spec coverage: Task 2 implements service-backed Hypridle and lid inhibitors, Task 3 gates laptop behavior and adds the recovery keybind, Task 4 implements the Eww frontend, Task 5 documents and verifies the feature.
- Placeholder scan: the plan contains no unfinished markers, vague error-handling instructions, or delegated test-writing steps.
- Type consistency: backend state uses `idle_inhibited` and `display` consistently; display modes are `normal`, `external`, and `headless`; CLI actions are `normal`, `external`, `headless`, `restore`, and `status`.
