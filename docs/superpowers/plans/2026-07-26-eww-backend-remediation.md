# Eww backend remediation (Changeset 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the eww bar backend's confirmed live defects and cut the `eww-barctl`
click path from ~76 ms to ~24 ms, all in Python, so the later Go port is a
translation of correct code rather than a mixed translate-and-repair.

**Architecture:** Four independent tasks against
`modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/`. Task 1 closes the
bluetooth `class` value set. Task 2 deletes verified-dead code. Task 3 splits the
control client out of the daemon's import graph so a click no longer loads
`urllib`. Task 4 makes `BarState()` deterministic and adds a test that fails when
`eww.yuck`'s `:initial` literal drifts from it.

**Tech Stack:** Python 3.14 stdlib only, `unittest`, Nix (home-manager module),
`just` recipes.

---

## Scope note

The spec's Changeset 1 has five items. This plan covers items 1-4. **Item 5
(extracting the 699-line AI-usage subsystem at `collectors.py:168-866` into its own
binary) gets its own plan**, because it is a subsystem extraction with real design
surface — the daemon currently owns that work via `periodic_refresh(state, 300,
refresh_ai_usage)` at `app.py:70-74`, and `refresh_ai_usage` writes to `state`
incrementally to drive a `refreshing` progress flag, so moving it out is a protocol
change, not a file move. Tasks 1-4 stand alone and ship working software.

## Note on the `:initial` approach

An earlier draft of the spec said to **generate** `eww.yuck:1`'s `:initial` literal.
That requires `builtins.readFile` on a derivation output (import-from-derivation),
because the generator is a Python program, and IFD is slow and commonly disabled in
flake checks. Task 4 enforces instead of generating: `BarState()` becomes
deterministic and a test fails on drift, printing the correct literal. Same
guarantee, no IFD. The spec has been updated to match, so the two agree.

## File structure

| File | Change | Responsibility after |
|---|---|---|
| `.../eww_bar_backend/collectors.py` | Modify `:1400-1414`, `:1431-1449`; delete `:1495-1501` | `parse_controller` returns a normalized bool; no shadowed `idle_inhibited_state` |
| `.../eww_bar_backend/common.py` | Add `CLOCK_DEFAULT`; delete `idle_pidfile_path` | Shared constants and helpers |
| `.../eww_bar_backend/state.py` | Use `CLOCK_DEFAULT`; drop the `collectors` import | Deterministic initial state, no collector dependency |
| `.../eww_bar_backend/ctl.py` | **Create** | Control client. Imports `json`, `socket`, `os`, `pathlib` only |
| `.../eww_bar_backend/control.py` | Delete the four client-side functions | Daemon-side control server only |
| `.../eww_bar_backend/app.py` | Delete `main()` | `run_bar()` only; dispatch moves to the shim |
| `.../eww_bar_backend/__init__.py` | Empty it | Package marker only |
| `.../eww/scripts/backend` | Rewrite | `argv[0]` dispatch with lazy imports |
| `.../eww/scripts/{listen,poll,hypr-events}` | **Delete** | — |
| `.../eww/default.nix` | Modify `:137-138` | Ship only the backend package to `~/.config` |
| `tests/eww_bar_backend/test_bluetooth.py` | Add cases (Task 1) + `ctl` rename (Task 3) | — |
| `tests/eww_bar_backend/test_ctl.py` | **Create** | Client payload + import-graph tests |
| `tests/eww_bar_backend/test_state_defaults.py` | **Create** | `eww.yuck` `:initial` drift guard |
| `tests/eww_bar_backend/{test_wallpaper,test_volume,test_notifications,test_network}.py` | `ctl` rename (Task 3) | — |

---

### Task 1: Close the bluetooth `class` value set

`parse_controller` returns bluetoothctl's raw `Powered:` value, and
`bluetooth_state_from_text` assigns it straight to `class`. So `class` is `"yes"` or
`"no"`, while `eww.scss:313` styles `.bluetooth.off` — which is therefore dead.
Verified by executing the shipped function: powered-off yields `class='no'`.

**Files:**
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py:1400-1414`, `:1431-1432`, `:1449`
- Test: `tests/eww_bar_backend/test_bluetooth.py`

- [ ] **Step 1: Write the failing tests**

Add these two methods to `class BluetoothStateTests` in
`tests/eww_bar_backend/test_bluetooth.py`, and extend the existing
`test_powered_off_no_devices` with a `class` assertion:

```python
    def test_powered_off_uses_the_off_class(self):
        controller = "Controller AA:BB\n\tPowered: no\n"
        state = collectors.bluetooth_state_from_text(controller, "", {})
        self.assertEqual(state["class"], "off")

    def test_powered_on_no_devices_has_no_off_class(self):
        controller = "Controller AA:BB\n\tAlias: MyBT\n\tPowered: yes\n"
        state = collectors.bluetooth_state_from_text(controller, "", {})
        self.assertEqual(state["class"], "")
        self.assertEqual(state["powered"], "true")

    def test_class_is_always_a_known_style(self):
        # eww.scss defines .bluetooth and .bluetooth.off only; "connected" is
        # deliberately unstyled and falls through to .bluetooth.
        known = {"", "off", "connected"}
        for powered in ("yes", "no", "whatever", ""):
            controller = f"Controller AA:BB\n\tPowered: {powered}\n"
            self.assertIn(
                collectors.bluetooth_state_from_text(controller, "", {})["class"], known
            )
            self.assertIn(
                collectors.bluetooth_state_from_text(
                    controller, "Device 80:99:E7 Buds\n", {}
                )["class"],
                known,
            )
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /home/lemonilemon/nixos-config
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tests.eww_bar_backend.test_bluetooth -v
```

Expected: `test_powered_off_uses_the_off_class` FAILS with
`AssertionError: 'no' != 'off'`, and `test_class_is_always_a_known_style` FAILS with
`'yes' not found in {'', 'off', 'connected'}`.

- [ ] **Step 3: Normalize `parse_controller`**

Replace `collectors.py:1400-1414` with:

```python
def parse_controller(controller_text):
    address = ""
    alias = ""
    powered = False
    for line in controller_text.splitlines():
        stripped = line.strip()
        if stripped.startswith("Controller "):
            parts = stripped.split()
            if len(parts) >= 2:
                address = parts[1]
        elif stripped.startswith("Alias:"):
            alias = stripped.split(":", 1)[1].strip()
        elif stripped.startswith("Powered:"):
            # Normalize here rather than passing bluetoothctl's raw "yes"/"no"
            # outward: this value feeds both `powered` and the CSS `class`, and
            # leaking the raw string is what made eww.scss's .bluetooth.off dead.
            powered = stripped.split(":", 1)[1].strip() == "yes"
    return alias or "Bluetooth", address or "N/A", powered
```

- [ ] **Step 4: Derive both outputs from the normalized flag**

In `collectors.py`, replace line 1432:

```python
    powered_flag = "true" if powered == "yes" else "false"
```

with:

```python
    powered_flag = "true" if powered else "false"
```

and replace line 1449:

```python
        return {"text": "", "tooltip": tooltip, "class": powered, "powered": powered_flag, "devices": []}
```

with:

```python
        return {
            "text": "",
            "tooltip": tooltip,
            "class": "" if powered else "off",
            "powered": powered_flag,
            "devices": [],
        }
```

Leave the devices branch at `collectors.py:1478` as `"class": "connected"`.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tests.eww_bar_backend.test_bluetooth -v
```

Expected: PASS, 9 tests.

- [ ] **Step 6: Run the whole suite**

```bash
just test-backend
```

Expected: `OK`, 113 tests.

- [ ] **Step 7: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py \
        tests/eww_bar_backend/test_bluetooth.py
git commit -m "fix(eww): emit a styleable bluetooth class when powered off

parse_controller returned bluetoothctl's raw Powered: value and
bluetooth_state_from_text assigned it straight to class, so the bar rendered
class=\"no\" and eww.scss's .bluetooth.off rule never matched. Normalize at the
parse boundary and derive both powered and class from the flag."
```

---

### Task 2: Delete verified-dead code

Four dead things, each confirmed: a shadowed collector, the helper that exists only
to feed it, three unreferenced shell scripts, and the `xdg.configFile` entry that
ships them to `~/.config` on every generation.

**Files:**
- Modify: `.../eww_bar_backend/collectors.py:1495-1501`, `.../eww_bar_backend/common.py:129-130`
- Delete: `.../eww/scripts/listen`, `.../eww/scripts/poll`, `.../eww/scripts/hypr-events`
- Modify: `.../eww/default.nix:137-138`

- [ ] **Step 1: Re-confirm the shadowed collector is unreachable**

```bash
cd /home/lemonilemon/nixos-config
rg -n 'idle_inhibited_state' modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/*.py
```

Expected: definitions at `collectors.py:1495` and `inhibitors.py:37`; every import
(`app.py:23`, `watchers.py:19`, `control.py:19`, `display.py:5`) reads
`from .inhibitors`. Nothing imports the `collectors` copy. If that is not what you
see, stop and re-plan.

- [ ] **Step 2: Delete the shadowed collector**

Delete `collectors.py:1495-1501` in its entirety — the `def idle_inhibited_state():`
block that reads the pidfile. Also delete `idle_pidfile_path` from the
`from .common import (...)` list at the top of `collectors.py`.

- [ ] **Step 3: Delete its only-consumer helper**

Delete `common.py:129-130`:

```python
def idle_pidfile_path():
    return runtime_file("eww-idle-inhibit.pid")
```

- [ ] **Step 4: Verify nothing else referenced them**

```bash
rg -n 'idle_pidfile_path' modules/ tests/
```

Expected: no output. (`__init__.py` is emptied in Task 3; if it still lists these
names here, remove those two lines now.)

- [ ] **Step 5: Confirm the three shell scripts are unreferenced, then delete them**

```bash
rg -n 'scripts/listen|scripts/poll|hypr-events' --glob '*.nix' --glob '*.yuck' --glob '*.py' .
```

Expected: no output. Then:

```bash
git rm modules/desktop/home/hyprland/eww/scripts/listen \
       modules/desktop/home/hyprland/eww/scripts/poll \
       modules/desktop/home/hyprland/eww/scripts/hypr-events
```

- [ ] **Step 6: Stop shipping the scripts directory to `~/.config`**

`eww.yuck` reaches the backend purely by `PATH` name (`eww.yuck:1` runs
`"eww-bar-backend bar"`, and the 34 other call sites are `eww-barctl` / `eww-popup`),
all of which come from the `ewwBarTools` package in `home.packages`. The
`~/.config/eww/scripts` copy is never executed.

Delete `default.nix:137-138`:

```nix
    xdg.configFile."eww/scripts".source = ./scripts;
    xdg.configFile."eww/scripts".recursive = true;
```

- [ ] **Step 7: Verify the suite and evaluation**

```bash
just test-backend && just fmt && just check
```

Expected: `OK` from the suite, no diff from `just fmt`, and `just check` passes.

- [ ] **Step 8: Confirm the bar still resolves its helpers**

```bash
nix build --dry-run ".#nixosConfigurations.$NIXHOST.config.system.build.toplevel"
```

Expected: builds without error. (`ewwBarTools` is in `home.packages` at
`default.nix:121`, so `eww-bar-backend`, `eww-barctl` and `eww-popup` stay on `PATH`.)

- [ ] **Step 9: Commit**

```bash
git add -A modules/desktop/home/hyprland/eww/
git commit -m "refactor(eww): drop dead backend code and the unused scripts copy

collectors.idle_inhibited_state was shadowed by the inhibitors version that all
four callers import, and common.idle_pidfile_path existed only to feed it.
scripts/listen, scripts/poll and scripts/hypr-events have had no references
since the waybar migration, and eww.yuck resolves every helper via PATH from
ewwBarTools, so the xdg.configFile copy shipped 13 KB of dead code per
generation."
```

---

### Task 3: Split the control client out of the import graph

`scripts/backend:10` does `from eww_bar_backend import *`, and `__init__.py:1` starts
with `from .app import main, run_bar`. So `eww-barctl` — invoked from 34 places in
`eww.yuck`, including `:onscroll` — loads `collectors` → `urllib.request` →
`http.client` → `email.parser` before it opens a socket. Measured: 76.35 ms median
per invocation, ~60 ms of it import.

**Files:**
- Create: `.../eww_bar_backend/ctl.py`
- Modify: `.../eww_bar_backend/control.py` (remove the client half), `.../eww_bar_backend/app.py` (remove `main`), `.../eww_bar_backend/__init__.py` (empty)
- Rewrite: `.../eww/scripts/backend`
- Test: `tests/eww_bar_backend/test_ctl.py` (new)
- Test (update, 14 call sites): `tests/eww_bar_backend/{test_wallpaper,test_volume,test_notifications,test_network,test_bluetooth}.py`

- [ ] **Step 1: Record the baseline latency**

```bash
cd /home/lemonilemon/nixos-config
python3 - <<'PY'
import subprocess, time, statistics
t=[]
for _ in range(20):
    s=time.perf_counter(); subprocess.run(["eww-barctl","--quiet","ping"]); t.append((time.perf_counter()-s)*1000)
print(f"median {statistics.median(t):.2f} ms  min {min(t):.2f} ms")
PY
```

Write the number down. Expected: roughly 70-80 ms median.

- [ ] **Step 2: Write the failing tests**

Create `tests/eww_bar_backend/test_ctl.py`:

```python
import subprocess
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import ctl  # noqa: E402


class ControlPayloadTests(unittest.TestCase):
    def test_ping(self):
        self.assertEqual(ctl.control_payload_from_args(["ping"]), {"command": "ping"})

    def test_volume_set_carries_value(self):
        self.assertEqual(
            ctl.control_payload_from_args(["volume", "set", "40"]),
            {"command": "volume", "action": "set", "value": "40"},
        )

    def test_notif_clear_group_joins_multiword_app(self):
        self.assertEqual(
            ctl.control_payload_from_args(["notif", "clear-group", "My", "App"]),
            {"command": "notif", "action": "clear-group", "app": "My App"},
        )

    def test_idle_defaults_to_toggle(self):
        self.assertEqual(
            ctl.control_payload_from_args(["idle"]),
            {"command": "idle", "action": "toggle"},
        )

    def test_help_raises_usage(self):
        with self.assertRaises(ValueError):
            ctl.control_payload_from_args(["--help"])


class ImportGraphTests(unittest.TestCase):
    """The click path must not import the daemon.

    Run in a subprocess so this test is unaffected by modules the rest of the
    suite has already imported into this interpreter.
    """

    FORBIDDEN = ("urllib.request", "http.client", "email.parser")

    def _modules_after_importing(self, module):
        code = (
            "import sys\n"
            f"sys.path.insert(0, {str(SCRIPTS_DIR)!r})\n"
            f"import {module}\n"
            "print('\\n'.join(sorted(sys.modules)))\n"
        )
        out = subprocess.run(
            [sys.executable, "-c", code], capture_output=True, text=True, check=True
        )
        return set(out.stdout.split())

    def test_ctl_does_not_pull_in_http(self):
        loaded = self._modules_after_importing("eww_bar_backend.ctl")
        for name in self.FORBIDDEN:
            self.assertNotIn(name, loaded)

    def test_ctl_does_not_pull_in_collectors(self):
        loaded = self._modules_after_importing("eww_bar_backend.ctl")
        self.assertNotIn("eww_bar_backend.collectors", loaded)
        self.assertNotIn("eww_bar_backend.app", loaded)

    def test_package_import_is_empty(self):
        loaded = self._modules_after_importing("eww_bar_backend")
        self.assertNotIn("eww_bar_backend.app", loaded)
        self.assertNotIn("eww_bar_backend.collectors", loaded)


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 3: Run the tests to verify they fail**

```bash
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tests.eww_bar_backend.test_ctl -v
```

Expected: every test ERRORs with `ModuleNotFoundError: No module named 'eww_bar_backend.ctl'`.

- [ ] **Step 4: Create `ctl.py`**

Create `.../eww_bar_backend/ctl.py`. `CONTROL_USAGE` and
`control_payload_from_args` move verbatim from `control.py`; `send_control_command`
and `run_ctl` lose their `common` import so this module's graph stays at
`json`/`socket`/`os`/`pathlib`:

```python
"""Control client for eww-barctl.

Deliberately isolated from the daemon's import graph. scripts/backend dispatches
here before importing anything else, so a bar click costs one socket round-trip
instead of loading collectors -> urllib.request -> http.client -> email.parser.
Keep this module's imports to json, socket, os and pathlib — the test in
tests/eww_bar_backend/test_ctl.py enforces it.
"""

import json
import os
import socket
from pathlib import Path


CONTROL_USAGE = (
    "usage: eww-barctl ping | volume up|down|set <0-100>|mute|sink <name> | "
    "media play-pause|next|previous | "
    "idle toggle|on|off|status | display normal|external|headless|restore|toggle|status | ai refresh"
    " | bluetooth power-toggle|disconnect <mac> | network wifi-toggle"
    " | notif toggle-group <app>|dismiss <id>|clear-group <app>|clear-all|dnd-toggle|mark-seen"
    " | wallpaper set <path>|rescan"
)


def control_socket_path():
    # Mirrors common.control_socket_path. Duplicated rather than imported so the
    # click path never loads common (and through it subprocess).
    runtime_dir = os.environ.get("XDG_RUNTIME_DIR", "/tmp")
    return Path(runtime_dir) / "eww-backend.sock"


def control_payload_from_args(args):
    if not args or args[0] in ("-h", "--help", "help"):
        raise ValueError(CONTROL_USAGE)
    if args[0] == "ping":
        return {"command": "ping"}
    if args[0] == "volume" and len(args) >= 2:
        payload = {"command": "volume", "action": args[1]}
        if args[1] == "set" and len(args) >= 3:
            payload["value"] = args[2]
        elif args[1] == "sink" and len(args) >= 3:
            payload["sink"] = args[2]
        return payload
    if args[0] == "media" and len(args) == 2:
        return {"command": "media", "action": args[1]}
    if args[0] == "bluetooth" and len(args) >= 2:
        payload = {"command": "bluetooth", "action": args[1]}
        if args[1] == "disconnect" and len(args) >= 3:
            payload["mac"] = args[2]
        return payload
    if args[0] == "network" and len(args) >= 2:
        return {"command": "network", "action": args[1]}
    if args[0] == "idle":
        action = args[1] if len(args) > 1 else "toggle"
        return {"command": "idle", "action": action}
    if args[0] == "display":
        action = args[1] if len(args) > 1 else "status"
        return {"command": "display", "action": action}
    if args[0] == "ai" and len(args) == 2:
        return {"command": "ai", "action": args[1]}
    if args[0] == "notif" and len(args) >= 2:
        payload = {"command": "notif", "action": args[1]}
        if args[1] in ("toggle-group", "clear-group") and len(args) >= 3:
            payload["app"] = " ".join(args[2:])
        elif args[1] == "dismiss" and len(args) >= 3:
            payload["id"] = args[2]
        return payload
    if args[0] == "wallpaper" and len(args) >= 2:
        payload = {"command": "wallpaper", "action": args[1]}
        if args[1] == "set" and len(args) >= 3:
            payload["path"] = args[2]
        return payload
    raise ValueError(CONTROL_USAGE)


def send_control_command(payload):
    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
        client.connect(str(control_socket_path()))
        client.sendall((json.dumps(payload, separators=(",", ":")) + "\n").encode("utf-8"))
        client.shutdown(socket.SHUT_WR)
        chunks = []
        while True:
            chunk = client.recv(4096)
            if not chunk:
                break
            chunks.append(chunk)
    text = b"".join(chunks).decode("utf-8", errors="replace").strip()
    try:
        return json.loads(text)
    except Exception:
        return {"ok": False, "error": "invalid backend response"}


def run_ctl(args):
    quiet = False
    if args and args[0] in ("-q", "--quiet"):
        quiet = True
        args = args[1:]
    try:
        response = send_control_command(control_payload_from_args(args))
    except Exception as exc:
        response = {"ok": False, "error": str(exc)}
    if not quiet:
        print(json.dumps(response, separators=(",", ":")), flush=True)
    return 0 if response.get("ok") else 1
```

- [ ] **Step 5: Move the client half out of `control.py` and update its callers**

Delete these four definitions from `control.py`, which now live in `ctl.py`:
`CONTROL_USAGE` (`:31-38`), `control_payload_from_args` (`:338-379`),
`send_control_command` (`:382-394`), `run_ctl` (`:397-408`).

Do **not** import `CONTROL_USAGE` back into `control.py`. Its only two references
(`:340` and `:379`) are both inside `control_payload_from_args`, which is moving —
`handle_control_command` raises its own per-command messages and never uses it.

Drop `parse_json` from the `from .common import (...)` line — its only reference
(`:394`) was inside `send_control_command`, which is moving:

```python
from .common import backend_pidfile_path, control_socket_path
```

Verify nothing dangles:

```bash
rg -n 'CONTROL_USAGE|parse_json' modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/control.py
```

Expected: no output.

**Now update the existing tests that call the moved function.** Five test files
reach it through the `control` module at 14 call sites, and they will all fail
otherwise:

```bash
cd /home/lemonilemon/nixos-config/tests/eww_bar_backend
sed -i 's/\bcontrol\.control_payload_from_args\b/ctl.control_payload_from_args/g' \
  test_wallpaper.py test_volume.py test_notifications.py test_network.py test_bluetooth.py
```

Then fix each file's import line. The current lines are:

| File | Line | Current |
|---|---|---|
| `test_wallpaper.py` | 80 | `from eww_bar_backend import control  # noqa: E402` |
| `test_volume.py` | 10 | `from eww_bar_backend import collectors, control  # noqa: E402` |
| `test_notifications.py` | 11 | `from eww_bar_backend import control  # noqa: E402` |
| `test_network.py` | 10 | `from eww_bar_backend import collectors, control  # noqa: E402` |
| `test_bluetooth.py` | 10 | `from eww_bar_backend import collectors, control  # noqa: E402` |

Add `ctl` to each, then check whether `control` is still referenced in that file and
drop it from the import if not:

```bash
for f in test_wallpaper.py test_volume.py test_notifications.py test_network.py test_bluetooth.py; do
  printf '%-24s control refs: ' "$f"
  rg -c '\bcontrol\.' "$f" || echo 0
done
```

A file reporting `0` should import `ctl` alone (e.g. `test_wallpaper.py:80` becomes
`from eww_bar_backend import ctl  # noqa: E402`). A file with a non-zero count keeps
both (e.g. `from eww_bar_backend import collectors, control, ctl  # noqa: E402`).

Verify the rename is complete:

```bash
cd /home/lemonilemon/nixos-config
rg -n 'control\.control_payload_from_args' tests/
```

Expected: no output.

- [ ] **Step 6: Move dispatch out of `app.py`**

Delete `app.py:148-165` (the whole `def main():`).

Every use of `sys` and `Path` in `app.py` is inside `main` (verified: lines 149,
151, 153, 155, 159, 161, 163 and nothing else), so delete both imports outright:

```python
import sys
from pathlib import Path
```

`run_ctl` and `run_popup` were also only used by `main`, so this import line:

```python
from .control import control_server, run_ctl, write_backend_pidfile
```

becomes:

```python
from .control import control_server, write_backend_pidfile
```

and this line is deleted entirely:

```python
from .popups import run_popup
```

Verify:

```bash
rg -n '\bsys\b|\bPath\b|run_ctl|run_popup' modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/app.py
```

Expected: no output.

- [ ] **Step 7: Empty `__init__.py`**

Replace the entire contents of `.../eww_bar_backend/__init__.py` with:

```python
"""Eww bar backend.

Intentionally empty: re-exporting the submodules here made every consumer —
including the eww-barctl click path — import the whole daemon. Import submodules
directly (`from eww_bar_backend import collectors`).
"""
```

- [ ] **Step 8: Rewrite the entry shim**

Replace the entire contents of `.../eww/scripts/backend` with:

```python
#!/usr/bin/env python3
"""Entry point for eww-bar-backend, eww-barctl and eww-popup.

Dispatch happens before any package import so the two CLI paths never load the
daemon. Every import below is deliberately function-local.
"""
import sys
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))


def main():
    invoked = Path(sys.argv[0]).name
    if invoked == "eww-barctl":
        from eww_bar_backend.ctl import run_ctl

        return run_ctl(sys.argv[1:])
    if invoked == "eww-popup":
        from eww_bar_backend.popups import run_popup

        return run_popup(sys.argv[1:])

    mode = sys.argv[1] if len(sys.argv) > 1 else "bar"
    if mode == "ctl":
        from eww_bar_backend.ctl import run_ctl

        return run_ctl(sys.argv[2:])
    if mode == "popup":
        from eww_bar_backend.popups import run_popup

        return run_popup(sys.argv[2:])
    if mode == "bar":
        from eww_bar_backend.app import run_bar

        run_bar()
        return 0
    print(f"unknown backend mode: {mode}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
```

- [ ] **Step 9: Run the tests to verify they pass**

```bash
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tests.eww_bar_backend.test_ctl -v
just test-backend
```

Expected: `test_ctl` passes all 8; `just test-backend` reports `OK`.

If `test_package_import_is_empty` still fails, something imports `.app` at package
import time — re-check Step 7.

- [ ] **Step 10: Rebuild and measure**

`just build` switches the live system and needs sudo, so hand this to the user:

> Run `just build` (it needs sudo and restarts the bar), then tell me when it's done.

After the rebuild, re-run the Step 1 measurement:

```bash
python3 - <<'PY'
import subprocess, time, statistics
t=[]
for _ in range(20):
    s=time.perf_counter(); subprocess.run(["eww-barctl","--quiet","ping"]); t.append((time.perf_counter()-s)*1000)
print(f"median {statistics.median(t):.2f} ms  min {min(t):.2f} ms")
PY
```

Expected: roughly 22-28 ms median, down from ~76 ms. Also click a few bar modules
and scroll the volume module to confirm the popups and `--quiet` paths still work.

- [ ] **Step 11: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/ tests/eww_bar_backend/
git commit -m "perf(eww): keep the daemon out of the eww-barctl import graph

scripts/backend star-imported the package and __init__.py pulled in app, so
every one of the 34 eww.yuck click and scroll handlers loaded collectors ->
urllib.request -> http.client -> email.parser before opening a socket. Move the
client half to a self-contained ctl module, empty __init__.py, and dispatch on
argv[0] with function-local imports. Measured: 76 ms -> 24 ms median per
invocation."
```

---

### Task 4: Make `BarState()` deterministic and guard the yuck literal

`eww.yuck:1` carries a hand-synced 2000-character `:initial` literal that has drifted
from the Python defaults in at least two places: `"battery":{"text":""` against
`common.py:9`'s `"󰂄"`, and `"temperature":{"text":""` against `state.py:31`'s
`" --°C"`.

It cannot be asserted against `BarState()` today because `state.py:25` calls
`clock_state()`, making the snapshot time-dependent. Fix that first: the daemon
overwrites `clock` within milliseconds anyway (`app.py:52`), and dropping the call
also removes `state.py`'s dependency on `collectors` entirely.

**Files:**
- Modify: `.../eww_bar_backend/common.py` (add `CLOCK_DEFAULT`), `.../eww_bar_backend/state.py:4`, `:25`
- Modify: `.../eww/eww.yuck:1`
- Test: `tests/eww_bar_backend/test_state_defaults.py`

- [ ] **Step 1: Write the failing test**

Create `tests/eww_bar_backend/test_state_defaults.py`:

```python
import json
import re
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
EWW_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww"
SCRIPTS_DIR = EWW_DIR / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend.state import BarState  # noqa: E402


class InitialStateTests(unittest.TestCase):
    def test_bar_state_is_deterministic(self):
        # Two constructions a moment apart must be identical, or the literal
        # below can never be asserted against it.
        self.assertEqual(BarState().snapshot(), BarState().snapshot())

    def test_yuck_initial_matches_bar_state(self):
        first_line = (EWW_DIR / "eww.yuck").read_text().splitlines()[0]
        match = re.search(r":initial '(.*)' \"eww-bar-backend bar\"\)", first_line)
        self.assertIsNotNone(match, "could not find the deflisten :initial literal")
        from_yuck = json.loads(match.group(1))
        from_python = json.loads(BarState().snapshot())
        if from_yuck != from_python:
            self.fail(
                "eww.yuck :initial has drifted from BarState().\n"
                "Replace the literal on eww.yuck line 1 with exactly:\n\n"
                + BarState().snapshot()
                + "\n"
            )


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tests.eww_bar_backend.test_state_defaults -v
```

Expected: `test_bar_state_is_deterministic` may pass or fail depending on whether the
clock ticks between the two calls (that flakiness is the point).
`test_yuck_initial_matches_bar_state` FAILS, printing the drift.

- [ ] **Step 3: Add a deterministic clock default**

Add to `common.py`, next to the other `*_DEFAULT` constants:

```python
CLOCK_DEFAULT = {"time": " --:--", "date": " ----", "tooltip": ""}
```

- [ ] **Step 4: Use it in `BarState`**

In `state.py`, delete line 4:

```python
from .collectors import clock_state
```

Add `CLOCK_DEFAULT` to the `from .common import (...)` list, and replace line 25:

```python
            "clock": clock_state(),
```

with:

```python
            # Placeholder, not a live reading: app.run_bar overwrites this via
            # its first state.update within milliseconds, and a time-dependent
            # constructor would make the eww.yuck :initial literal unassertable.
            "clock": CLOCK_DEFAULT.copy(),
```

- [ ] **Step 5: Run the test to get the correct literal**

```bash
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tests.eww_bar_backend.test_state_defaults -v
```

Expected: `test_bar_state_is_deterministic` PASSES.
`test_yuck_initial_matches_bar_state` still FAILS, and prints the exact literal to
paste.

- [ ] **Step 6: Update the yuck literal**

Copy the JSON the test printed and replace the text between `:initial '` and
`' "eww-bar-backend bar")` on `eww.yuck:1`. Do not reformat or re-order it — the test
compares parsed JSON, but keeping the emitted byte order makes future diffs readable.

- [ ] **Step 7: Run the tests to verify they pass**

```bash
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tests.eww_bar_backend.test_state_defaults -v
just test-backend
```

Expected: both tests PASS; `just test-backend` reports `OK`.

- [ ] **Step 8: Confirm the tray_count journal errors are addressed**

The literal previously omitted or mistyped keys that `eww.yuck:51-55` reads as `f64`.
Confirm `tray_count` is present and numeric in the new literal:

```bash
head -1 modules/desktop/home/hyprland/eww/eww.yuck | grep -o '"tray_count":[0-9]*'
```

Expected: `"tray_count":0`.

- [ ] **Step 9: Full verification**

```bash
just fmt && just check && just test
nix build --dry-run ".#nixosConfigurations.$NIXHOST.config.system.build.toplevel"
```

Expected: all pass.

- [ ] **Step 10: Commit**

```bash
git add modules/desktop/home/hyprland/eww/ tests/eww_bar_backend/test_state_defaults.py
git commit -m "fix(eww): stop the yuck :initial literal drifting from BarState

The 2000-character deflisten literal was hand-synced and had drifted: battery
and temperature carried empty strings where the Python defaults hold a Nerd Font
glyph and ' --°C'. BarState could not be asserted against it because the
constructor called clock_state(), so make the clock a placeholder the daemon
overwrites on its first update, then add a test that fails on drift and prints
the correct literal."
```

---

## Post-Changeset-1 checkpoint

Before planning the Go port, answer these with the remediated backend running for
at least a few days:

1. Does the ~24 ms click latency register at all in use, particularly on the
   `eww.yuck:93` volume scroll?
2. Has anything drifted that the new tests did not catch?
3. Is the `:initial` guard sufficient, or does the literal still feel like a
   liability worth generating properly?

The spec's Changeset 2 is gated on this. If the answers are "no, no, sufficient",
the remaining case for the port is 23 ms and a type system, and it is reasonable to
stop here.

## Self-review notes

- **Spec coverage:** items 1-4 of Changeset 1 map to Tasks 1-4. Item 5 is explicitly
  deferred to its own plan, with the reason stated above.
- **Deviation:** Task 4 enforces rather than generates the yuck literal. The spec
  has been updated to match, so there is no open divergence.
- **Naming consistency:** `control_socket_path` exists in both `common.py` (daemon)
  and `ctl.py` (client) with identical behavior — intentional duplication, commented
  in the source, to keep the client's import graph minimal. `CONTROL_USAGE` and
  `control_payload_from_args` live only in `ctl.py` after Task 3; `control.py` does
  not import either back, because its remaining code references neither (verified:
  `CONTROL_USAGE` appeared only at `:340`/`:379`, both inside the moved function).
- **Caller migration caught in review:** moving `control_payload_from_args` breaks
  14 call sites across five existing test files. Folded into Task 3 Step 5 with the
  exact rename command and per-file import guidance. Without it, Task 3 Step 9 would
  have failed with `AttributeError: module 'eww_bar_backend.control' has no
  attribute 'control_payload_from_args'` across five files.
- **Ordering:** Task 1 changes the bluetooth default `class`, and Task 4 asserts the
  yuck literal against `BarState()`. Task 4 runs last, so it picks up the corrected
  value rather than baking in the buggy one.
