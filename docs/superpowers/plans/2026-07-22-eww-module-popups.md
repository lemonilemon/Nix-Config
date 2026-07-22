# Eww Module Popups Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the volume/bluetooth/network/battery bar modules' external-app and tooltip behaviors with interactive eww popups under one shared open/position/close interaction model.

**Architecture:** A single `eww-popup` Python helper (installed next to `eww-barctl`) owns popup lifecycle: it toggles one popup at a time and always opens a transparent `popup_backdrop` window (below the bar) first, so any outside click closes the popup. Each module's left click calls `eww-popup toggle <window> <screen>`; each popup is a top-right-anchored layer-shell window offset to sit under its module. Backend collectors gain structured state (volume dict + sinks, bluetooth devices, network wifi flag, battery health/power) and `eww-barctl` gains control verbs; popup widgets reuse the existing Catppuccin card design language.

**Tech Stack:** eww 0.6.0 (yuck + SCSS), Python 3 stdlib backend (`eww_bar_backend` package), `wpctl`/`pactl` (audio), `bluetoothctl`, `nmcli`, Nix Home Manager, `unittest`.

## Global Constraints

- eww version is `0.6.0`. Bars run per-monitor (`eDP-1`, `HDMI-A-1`); popups open on the clicked bar's screen via `--screen ${output}`.
- Backend is Python 3 **stdlib only** — no new Python dependencies. New logic is covered by subprocess-mocked `unittest` tests.
- Catppuccin Mocha SCSS variables already defined at the top of `eww.scss` (`$text`, `$yellow`, `$sapphire`, `$peach`, `$blue`, `$overlay0/1`, `$surface0/1`, `$red`, `$green`, …) — reuse them; do not add new hex literals except in `rgba(...)` tints matching the existing style.
- Popup windows are **single-instance** (open without `--id`, so window id == name). The helper opens `popup_backdrop` **before** the popup; all popups + backdrop use `:stacking "fg"`.
- The canonical popup window set (used by `eww-popup`) is exactly: `volume_popup`, `bluetooth_popup`, `network_popup`, `battery_popup`, `ai_usage_popup`, `display_mode_popup`; the backdrop is `popup_backdrop`.
- Any change to a state dict's shape MUST be mirrored in all three seed locations in the same commit: `scripts/eww_bar_backend/common.py` (defaults), `scripts/eww_bar_backend/state.py` (`BarState.__init__`), and the `deflisten bar_state :initial '{...}'` JSON in `eww.yuck` (line 1).
- Base directory for eww files: `modules/desktop/home/hyprland/eww/` (referred to below as `<eww>/`). Backend package: `<eww>/scripts/eww_bar_backend/`.
- Backend tests live in `tests/eww_bar_backend/`. Run a single file with `python3 tests/eww_bar_backend/test_<name>.py -v`; run all with `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py'` (root-level `discover` fails — the `tests/eww_bar_backend` package name collides with the real package).
- Build/verify: `just fmt` (`nix fmt`), `NIXHOST=laptop just test` (eval), `NIXHOST=laptop just build` (rebuild), then `systemctl --user restart eww-bar`. Screenshots for calibration: `grim /tmp/eww-shot.png` then read it.
- `pactl`, `wpctl`, `bluetoothctl`, `nmcli`, `kitty`, `hyprctl` are already on the eww runtime PATH. `blueman` is added in Task 1.
- Commit messages follow the repo's conventional style (`feat(eww): …`, `test(eww): …`). End each with the trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

---

## Task 1: `eww-popup` orchestration helper

**Files:**
- Create: `<eww>/scripts/eww_bar_backend/popups.py`
- Modify: `<eww>/scripts/eww_bar_backend/app.py` (dispatch on `argv[0] == "eww-popup"`)
- Modify: `<eww>/default.nix` (install `eww-popup`; add `blueman` to `runtimePackages`)
- Test: `tests/eww_bar_backend/test_popups.py`

**Interfaces:**
- Produces: the `eww-popup` CLI with `toggle <window-name> <screen>` and `close`; pure function `popup_eww_calls(command, args, open_ids) -> list[list[str]]` returning the ordered `eww` argument vectors; `parse_open_windows(text) -> set[str]`. Later tasks (2, 4, 6, 8, 10) wire module clicks to `eww-popup toggle …`.

- [ ] **Step 1: Write the failing test**

Create `tests/eww_bar_backend/test_popups.py`:

```python
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import popups  # noqa: E402


class ParseOpenWindowsTests(unittest.TestCase):
    def test_parses_ids_before_colon(self):
        text = "bar-eDP-1: bar\nvolume_popup: volume_popup\npopup_backdrop: popup_backdrop\n"
        self.assertEqual(
            popups.parse_open_windows(text),
            {"bar-eDP-1", "volume_popup", "popup_backdrop"},
        )

    def test_ignores_blank_and_malformed_lines(self):
        self.assertEqual(popups.parse_open_windows("\n   \ngarbage\n"), set())


class PopupCallsTests(unittest.TestCase):
    def test_toggle_open_closes_all_then_opens_backdrop_then_popup(self):
        calls = popups.popup_eww_calls("toggle", ["volume_popup", "eDP-1"], open_ids=set())
        self.assertEqual(
            calls,
            [
                ["close", *popups.POPUP_WINDOWS, popups.BACKDROP_WINDOW],
                ["open", "popup_backdrop", "--screen", "eDP-1"],
                ["open", "volume_popup", "--screen", "eDP-1"],
            ],
        )

    def test_toggle_when_already_open_only_closes(self):
        calls = popups.popup_eww_calls(
            "toggle", ["volume_popup", "eDP-1"], open_ids={"volume_popup"}
        )
        self.assertEqual(calls, [["close", *popups.POPUP_WINDOWS, popups.BACKDROP_WINDOW]])

    def test_close_closes_all(self):
        calls = popups.popup_eww_calls("close", [], open_ids=set())
        self.assertEqual(calls, [["close", *popups.POPUP_WINDOWS, popups.BACKDROP_WINDOW]])

    def test_unknown_window_rejected(self):
        with self.assertRaises(ValueError):
            popups.popup_eww_calls("toggle", ["nope_popup", "eDP-1"], open_ids=set())

    def test_toggle_requires_two_args(self):
        with self.assertRaises(ValueError):
            popups.popup_eww_calls("toggle", ["volume_popup"], open_ids=set())


class RunPopupTests(unittest.TestCase):
    def test_run_popup_toggle_invokes_eww_in_order(self):
        seen = []
        rc = popups.run_popup(
            ["toggle", "volume_popup", "HDMI-A-1"],
            active_windows_fn=lambda: "bar-eDP-1: bar\n",
            eww_fn=lambda args: seen.append(args),
        )
        self.assertEqual(rc, 0)
        self.assertEqual(
            seen,
            [
                ["close", *popups.POPUP_WINDOWS, popups.BACKDROP_WINDOW],
                ["open", "popup_backdrop", "--screen", "HDMI-A-1"],
                ["open", "volume_popup", "--screen", "HDMI-A-1"],
            ],
        )


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 tests/eww_bar_backend/test_popups.py -v`
Expected: FAIL — `ModuleNotFoundError`/`ImportError: cannot import name 'popups'`.

- [ ] **Step 3: Write minimal implementation**

Create `<eww>/scripts/eww_bar_backend/popups.py`:

```python
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python3 tests/eww_bar_backend/test_popups.py -v`
Expected: PASS (8 tests OK).

- [ ] **Step 5: Wire the CLI entry point**

Modify `<eww>/scripts/eww_bar_backend/app.py`. Add the import near the other `from .` imports:

```python
from .popups import run_popup
```

Replace the `main()` function's dispatch (currently keyed only on `eww-barctl`) with:

```python
def main():
    invoked = Path(sys.argv[0]).name
    if invoked == "eww-barctl":
        return run_ctl(sys.argv[1:])
    if invoked == "eww-popup":
        return run_popup(sys.argv[1:])

    mode = sys.argv[1] if len(sys.argv) > 1 else "bar"
    if mode == "bar":
        run_bar()
    elif mode == "ctl":
        return run_ctl(sys.argv[2:])
    elif mode == "popup":
        return run_popup(sys.argv[2:])
    else:
        print(f"unknown backend mode: {mode}", file=sys.stderr)
        return 1
    return 0
```

- [ ] **Step 6: Install `eww-popup` and add `blueman`**

Modify `<eww>/default.nix`. In the `ewwBarTools` `installPhase`, add an `eww-popup` copy next to the `eww-barctl` copy:

```nix
    installPhase = ''
      install -Dm755 ${./scripts/backend} $out/bin/eww-bar-backend
      cp -R ${./scripts/eww_bar_backend} $out/bin/eww_bar_backend
      cp $out/bin/eww-bar-backend $out/bin/eww-barctl
      cp $out/bin/eww-bar-backend $out/bin/eww-popup
      patchShebangs $out/bin
    '';
```

In `runtimePackages`, add `blueman` to the `with pkgs; [ … ]` list (alphabetically near `bluez`):

```nix
      bluez
      blueman
```

- [ ] **Step 7: Verify compile + eval**

Run: `python3 -m py_compile modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/*.py`
Expected: no output (success).

Run: `NIXHOST=laptop just test`
Expected: prints a `.drvPath` store path, no evaluation error.

- [ ] **Step 8: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/popups.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/app.py \
        modules/desktop/home/hyprland/eww/default.nix \
        tests/eww_bar_backend/test_popups.py
git commit -m "feat(eww): add eww-popup orchestration helper

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Backdrop window + retrofit existing popups

**Files:**
- Modify: `<eww>/eww.yuck` (add `popup_backdrop` window; re-route `ai_usage_popup` and `display_mode_popup` open/close through `eww-popup`)
- Modify: `<eww>/eww.scss` (add `.popup-backdrop`)

**Interfaces:**
- Consumes: `eww-popup toggle|close` from Task 1.
- Produces: `popup_backdrop` window + the outside-click-to-close behavior every later popup relies on.

- [ ] **Step 1: Add the backdrop window**

In `<eww>/eww.yuck`, add this window definition (place it just above the existing `(defwindow display_mode_popup …`):

```lisp
(defwindow popup_backdrop
  :monitor 0
  :geometry (geometry
    :x "0"
    :y "44px"
    :width "100%"
    :height "100%"
    :anchor "top center")
  :stacking "fg"
  :exclusive false
  :focusable false
  :namespace "eww-popup-backdrop"
  (eventbox :class "popup-backdrop" :onclick "eww-popup close"
    (box :hexpand true :vexpand true)))
```

- [ ] **Step 2: Route the AI-usage popup through the helper**

In `<eww>/eww.yuck`, in the `ai-usage` button, replace the `:onclick` (currently `eww open --toggle ai_usage_popup --screen ${output} && eww-barctl --quiet ai refresh`) with:

```lisp
            :onclick "eww-popup toggle ai_usage_popup ${output} && eww-barctl --quiet ai refresh"
```

- [ ] **Step 3: Route the display-mode popup through the helper**

In the `idle-inhibitor` button, replace the `:onrightclick` value (currently `{laptop_controls == "true" ? "eww open --toggle display_mode_popup --screen ${output}" : "true"}`) with:

```lisp
            :onrightclick {laptop_controls == "true" ? "eww-popup toggle display_mode_popup ${output}" : "true"}
```

In the `display_mode_panel` widget's footer, replace the "Exit mode" button's `:onclick` (currently `eww-barctl --quiet display normal && eww close display_mode_popup`) with:

```lisp
        :onclick "eww-barctl --quiet display normal && eww-popup close"
```

- [ ] **Step 4: Style the backdrop**

In `<eww>/eww.scss`, add near the end of the file:

```scss
.popup-backdrop {
  background: transparent;
}
```

- [ ] **Step 5: Build and verify the mechanism**

Run: `just fmt`
Then: `NIXHOST=laptop just build && systemctl --user restart eww-bar`

Manual checks (both monitors):
- Left-click the AI-usage module → popup opens. Click empty desktop → popup closes.
- Left-click AI-usage to open, then click the AI-usage icon again → popup closes.
- Left-click AI-usage on the `HDMI-A-1` bar → popup opens on `HDMI-A-1`.
- (laptop) Right-click the idle inhibitor → display-mode popup opens; "Exit mode" and outside-click both close it.

Expected: no popup stays "stuck" open; only one popup open at a time.

- [ ] **Step 6: Commit**

```bash
git add modules/desktop/home/hyprland/eww/eww.yuck modules/desktop/home/hyprland/eww/eww.scss
git commit -m "feat(eww): add popup backdrop and route existing popups through eww-popup

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Volume backend (state dict + control verbs)

**Files:**
- Modify: `<eww>/scripts/eww_bar_backend/collectors.py` (`volume_state` → dict, add sink helpers)
- Modify: `<eww>/scripts/eww_bar_backend/control.py` (add `set_volume`, `toggle_mute`, `set_sink`; extend `volume` command + arg parsing + usage)
- Modify: `<eww>/scripts/eww_bar_backend/common.py` (add `VOLUME_DEFAULT`)
- Modify: `<eww>/scripts/eww_bar_backend/state.py` (seed volume with the dict)
- Modify: `<eww>/eww.yuck` (deflisten seed; bar label → `.text`)
- Test: `tests/eww_bar_backend/test_volume.py`

**Interfaces:**
- Consumes: `run_text`, `parse_json`, `VOLUME_DEFAULT` from `common`.
- Produces: `volume_state() -> {text, percent, muted, class, sinks:[{name, description, active}]}`; control verbs `volume set <0-100> | mute | sink <name>` (plus existing `up|down`), all returning the fresh volume dict. Task 4 consumes `bar_state.volume.{text,percent,muted,sinks}` and the control verbs.

- [ ] **Step 1: Write the failing test**

Create `tests/eww_bar_backend/test_volume.py`:

```python
import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, control  # noqa: E402


class VolumeStateTests(unittest.TestCase):
    def test_unmuted_state(self):
        state = collectors.volume_state_from_text("Volume: 0.45\n", sinks=[])
        self.assertEqual(state["percent"], 45)
        self.assertEqual(state["muted"], "false")
        self.assertNotEqual(state["text"], "")
        self.assertEqual(state["sinks"], [])

    def test_muted_state(self):
        state = collectors.volume_state_from_text("Volume: 0.80 [MUTED]", sinks=[])
        self.assertEqual(state["muted"], "true")
        self.assertEqual(state["class"], "muted")

    def test_no_reading_falls_back_to_default(self):
        state = collectors.volume_state_from_text("garbage", sinks=[])
        self.assertEqual(state["percent"], 0)


class SinkParsingTests(unittest.TestCase):
    def test_json_sinks_mark_default_and_fall_back_description(self):
        payload = (
            '[{"index":56,"name":"alsa_speaker","description":"Built-in Speaker"},'
            '{"index":83,"name":"bt_headset","description":""}]'
        )
        sinks = collectors.sinks_from_pactl_json(payload, "bt_headset")
        self.assertEqual(
            sinks,
            [
                {"name": "alsa_speaker", "description": "Built-in Speaker", "active": "false"},
                {"name": "bt_headset", "description": "bt_headset", "active": "true"},
            ],
        )

    def test_short_sinks_fallback(self):
        short = "56\talsa_speaker\tPipeWire\ts32le\tSUSPENDED\n83\tbt_headset\tPipeWire\ts16le\tRUNNING\n"
        sinks = collectors.sinks_from_pactl_short(short, "bt_headset")
        self.assertEqual(
            sinks,
            [
                {"name": "alsa_speaker", "description": "alsa_speaker", "active": "false"},
                {"name": "bt_headset", "description": "bt_headset", "active": "true"},
            ],
        )


class VolumeControlTests(unittest.TestCase):
    def _capture_run(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        return calls, fake_run

    def test_set_volume_clamps_and_formats_percent(self):
        calls, fake_run = self._capture_run()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "volume_state", return_value={"ok": 1}):
                control.set_volume("142.6")
        self.assertEqual(calls, [["wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", "100%"]])

    def test_toggle_mute(self):
        calls, fake_run = self._capture_run()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "volume_state", return_value={}):
                control.toggle_mute()
        self.assertEqual(calls, [["wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle"]])

    def test_set_sink(self):
        calls, fake_run = self._capture_run()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "volume_state", return_value={}):
                control.set_sink("bt_headset")
        self.assertEqual(calls, [["pactl", "set-default-sink", "bt_headset"]])

    def test_arg_parsing_for_new_verbs(self):
        self.assertEqual(
            control.control_payload_from_args(["volume", "set", "40"]),
            {"command": "volume", "action": "set", "value": "40"},
        )
        self.assertEqual(
            control.control_payload_from_args(["volume", "sink", "bt_headset"]),
            {"command": "volume", "action": "sink", "sink": "bt_headset"},
        )
        self.assertEqual(
            control.control_payload_from_args(["volume", "up"]),
            {"command": "volume", "action": "up"},
        )


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 tests/eww_bar_backend/test_volume.py -v`
Expected: FAIL — `AttributeError: module 'eww_bar_backend.collectors' has no attribute 'volume_state_from_text'` signature mismatch / `sinks_from_pactl_json` missing / `control.set_volume` missing.

- [ ] **Step 3: Add `VOLUME_DEFAULT` to common.py**

In `<eww>/scripts/eww_bar_backend/common.py`, after `MEDIA_DEFAULT`:

```python
VOLUME_DEFAULT = {"text": "", "percent": 0, "muted": "false", "class": "", "sinks": []}
```

- [ ] **Step 4: Rewrite the volume collector**

In `<eww>/scripts/eww_bar_backend/collectors.py`, import `VOLUME_DEFAULT` (add it to the existing `from .common import (…)` block). Replace the current `volume_state_from_text` and `volume_state` (near the end of the file) with:

```python
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
        return f" {volume}%"
    if volume < 70:
        return f" {volume}%"
    return f" {volume}%"


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
                    "description": description if isinstance(description, str) and description else name,
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
            {"name": name, "description": name, "active": "true" if name == default_name else "false"}
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
```

> Note: preserve the exact speaker glyphs already in the file — copy the three `f" {volume}%"` lines and the `"󰝟"` muted glyph verbatim from the previous `volume_state_from_text` when typing `volume_label_from_text`.

- [ ] **Step 5: Extend the control surface**

In `<eww>/scripts/eww_bar_backend/control.py`:

Update `CONTROL_USAGE` — change the `volume up|down` fragment to:

```python
    "usage: eww-barctl ping | volume up|down|set <0-100>|mute|sink <name> | "
```

Add these functions next to `adjust_volume`:

```python
def set_volume(value):
    try:
        level = max(0, min(100, int(round(float(value)))))
    except (TypeError, ValueError):
        raise ValueError("volume set requires a number 0-100")
    subprocess.run(
        ["wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", f"{level}%"],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return volume_state()


def toggle_mute():
    subprocess.run(
        ["wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle"],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return volume_state()


def set_sink(name):
    if not name:
        raise ValueError("volume sink requires a sink name")
    subprocess.run(
        ["pactl", "set-default-sink", name],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return volume_state()
```

Replace the `if command == "volume":` block in `handle_control_command` with:

```python
    if command == "volume":
        action = payload.get("action", "")
        if action in ("up", "down"):
            value = adjust_volume(action)
        elif action == "set":
            value = set_volume(payload.get("value"))
        elif action == "mute":
            value = toggle_mute()
        elif action == "sink":
            value = set_sink(payload.get("sink", ""))
        else:
            raise ValueError("volume action must be up, down, set, mute, or sink")
        state.update(volume=value)
        return {"ok": True, "command": "volume", "action": action, "volume": value}
```

Change `adjust_volume` to accept the action name directly (it currently takes `direction`); the only change is the parameter is passed as `action` above, and `adjust_volume("up"/"down")` already matches. Leave `adjust_volume`'s body as-is.

Replace the volume line in `control_payload_from_args` (currently `if args[0] == "volume" and len(args) == 2:`) with:

```python
    if args[0] == "volume" and len(args) >= 2:
        payload = {"command": "volume", "action": args[1]}
        if args[1] == "set" and len(args) >= 3:
            payload["value"] = args[2]
        elif args[1] == "sink" and len(args) >= 3:
            payload["sink"] = args[2]
        return payload
```

- [ ] **Step 6: Seed the new shape (state.py + deflisten)**

In `<eww>/scripts/eww_bar_backend/state.py`, add `VOLUME_DEFAULT` to the `from .common import (…)` line, and change `"volume": ""` to `"volume": VOLUME_DEFAULT.copy()`.

In `<eww>/eww.yuck` line 1, inside the `deflisten bar_state :initial '{…}'` JSON, replace `"volume":""` with:

```json
"volume":{"text":"","percent":0,"muted":"false","class":"","sinks":[]}
```

Still in `<eww>/eww.yuck`, in the volume `button`'s label (currently `(label :text {bar_state.volume})`), change it to:

```lisp
              (label :text {bar_state.volume.text})))
```

- [ ] **Step 7: Run tests + compile**

Run: `python3 tests/eww_bar_backend/test_volume.py -v`
Expected: PASS.

Run: `python3 -m py_compile modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/*.py`
Expected: no output.

- [ ] **Step 8: Build and verify the bar still renders**

Run: `just fmt && NIXHOST=laptop just build && systemctl --user restart eww-bar`
Manual: the volume module still shows the correct `%`/mute glyph; scrolling it still changes volume.
CLI sanity: `eww-barctl volume set 30` then `eww-barctl volume mute` twice — the bar reflects each change.

- [ ] **Step 9: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/control.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/common.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/state.py \
        modules/desktop/home/hyprland/eww/eww.yuck \
        tests/eww_bar_backend/test_volume.py
git commit -m "feat(eww): enrich volume state with percent/mute/sinks and control verbs

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Volume popup (widget + shared popup styles + bar wiring)

**Files:**
- Modify: `<eww>/eww.yuck` (shared `popup_header` widget; `volume_popup` window + `volume_panel`; volume module onclick)
- Modify: `<eww>/eww.scss` (shared `.popup*` classes + volume specifics)

**Interfaces:**
- Consumes: `eww-popup toggle volume_popup ${output}` (Task 1); `bar_state.volume.{text,percent,muted,sinks}` and `volume set|mute|sink` (Task 3).
- Produces: shared `popup_header` widget + `.popup*` SCSS base classes reused by Tasks 6/8/10.

- [ ] **Step 1: Add the shared header widget + volume widgets**

In `<eww>/eww.yuck`, add near the other `defwidget`s:

```lisp
(defwidget popup_header [icon title subtitle]
  (box :class "popup-header" :orientation "h" :space-evenly false
    (box :class "popup-mark" (label :text icon))
    (box :class "popup-title-stack" :orientation "v" :space-evenly false
      (label :class "popup-title" :halign "start" :text title)
      (label :class "popup-eyebrow" :halign "start" :text subtitle))))

(defwidget volume_panel []
  (box :class "popup accent-volume" :orientation "v" :space-evenly false
    (popup_header :icon "" :title "Volume" :subtitle "Output level & device")
    (box :class "popup-body" :orientation "v" :space-evenly false
      (label :class "volume-value" :halign "start"
        :text {bar_state.volume.muted == "true" ? "Muted" : "${bar_state.volume.percent}%"})
      (scale :class "popup-slider volume-slider"
        :value {bar_state.volume.percent} :min 0 :max 100
        :onchange "eww-barctl --quiet volume set {}")
      (button :class "popup-action volume-mute ${bar_state.volume.muted == 'true' ? 'active' : ''}"
        :onclick "eww-barctl --quiet volume mute"
        (box :orientation "h" :space-evenly false
          (label :class "popup-action-icon" :text {bar_state.volume.muted == "true" ? "󰝟" : ""})
          (label :class "popup-action-label" :halign "start"
            :text {bar_state.volume.muted == "true" ? "Unmute" : "Mute"})))
      (box :class "popup-section-label" :orientation "h" :space-evenly false
        (label :halign "start" :text "OUTPUT"))
      (box :class "volume-sinks" :orientation "v" :space-evenly false
        (for sink in {bar_state.volume.sinks}
          (button :class "popup-row volume-sink ${sink.active == 'true' ? 'active' : ''}"
            :onclick "eww-barctl --quiet volume sink ${sink.name}"
            (box :orientation "h" :space-evenly false
              (label :class "volume-sink-dot" :text {sink.active == "true" ? "" : ""})
              (label :class "volume-sink-name" :halign "start" :text {sink.description}))))))))
```

- [ ] **Step 2: Add the volume window**

In `<eww>/eww.yuck`, add near the other `defwindow`s:

```lisp
(defwindow volume_popup
  :monitor 0
  :geometry (geometry :x "70px" :y "50px" :width "300px" :height "240px" :anchor "top right")
  :stacking "fg"
  :exclusive false
  :focusable false
  :namespace "eww-volume"
  (volume_panel))
```

- [ ] **Step 3: Wire the volume module's click**

In `<eww>/eww.yuck`, in the volume `button` (inside the `session-island` eventbox), replace `:onclick "hyprctl dispatch exec 'kitty -- pulsemixer'"` with:

```lisp
            :onclick "eww-popup toggle volume_popup ${output}"
```

Leave the surrounding `(eventbox :onscroll "eww-barctl --quiet volume {}" …)` unchanged.

- [ ] **Step 4: Add shared popup + volume SCSS**

In `<eww>/eww.scss`, add (before the `.popup-backdrop` rule from Task 2):

```scss
/* --- Shared card for module popups (volume/bluetooth/network/battery) --- */
.popup {
  background: rgba(17, 17, 27, 0.985);
  border: 1px solid rgba(205, 214, 244, 0.14);
  border-radius: 12px;
  box-shadow: 0 20px 48px rgba(7, 8, 17, 0.58);
  color: $text;
}

.popup-header {
  border-bottom: 1px solid rgba(205, 214, 244, 0.07);
  padding: 14px 15px 13px;
}

.popup-mark {
  background: rgba(205, 214, 244, 0.08);
  border: 1px solid rgba(205, 214, 244, 0.14);
  border-radius: 8px;
  font-size: 15px;
  margin-right: 9px;
  min-height: 34px;
  min-width: 34px;
}

.popup-title { color: $text; font-size: 16px; font-weight: 800; }
.popup-eyebrow { color: $overlay1; font-size: 10px; margin-top: 1px; }
.popup-body { padding: 12px 14px 14px; }

.popup-section-label {
  color: $overlay0;
  font-size: 10px;
  font-weight: 800;
  margin: 12px 2px 4px;
}

.popup-row {
  border-radius: 7px;
  min-height: 30px;
  padding: 0 9px;
}
.popup-row:hover { background: rgba(49, 50, 68, 0.5); }

.popup-action {
  border-radius: 7px;
  color: $subtext0;
  margin-top: 8px;
  min-height: 32px;
  padding: 0 9px;
}
.popup-action:hover { background: rgba(49, 50, 68, 0.78); color: $text; }
.popup-action-icon { margin-right: 9px; min-width: 18px; }
.popup-action-label { font-weight: 700; }

.popup-footer-action {
  border-radius: 7px;
  color: $sapphire;
  font-size: 11px;
  font-weight: 800;
  margin-top: 10px;
  min-height: 28px;
  padding: 0 10px;
}
.popup-footer-action:hover { background: rgba(49, 50, 68, 0.6); }

.popup-empty { color: $overlay1; font-size: 11px; padding: 8px 2px; }

.popup-slider trough {
  background: rgba(69, 71, 90, 0.46);
  border-radius: 999px;
  min-height: 6px;
}
.popup-slider trough highlight { border-radius: 999px; min-height: 6px; }
.popup-slider trough slider {
  background: $text;
  border-radius: 999px;
  min-height: 14px;
  min-width: 14px;
  margin: -5px;
}

.popup-meter trough {
  background: rgba(69, 71, 90, 0.46);
  border-radius: 999px;
  min-height: 6px;
}
.popup-meter trough progress { border-radius: 999px; min-height: 6px; }

/* Accents */
.accent-volume .popup-mark { background: rgba(249, 226, 175, 0.11); border-color: rgba(249, 226, 175, 0.18); color: $yellow; }
.accent-bluetooth .popup-mark { background: rgba(116, 199, 236, 0.11); border-color: rgba(116, 199, 236, 0.18); color: $sapphire; }
.accent-network .popup-mark { background: rgba(250, 179, 135, 0.11); border-color: rgba(250, 179, 135, 0.18); color: $peach; }
.accent-battery .popup-mark { background: rgba(137, 180, 250, 0.11); border-color: rgba(137, 180, 250, 0.18); color: $blue; }

/* Volume specifics */
.volume-value { color: $text; font-size: 22px; font-weight: 900; margin-bottom: 4px; }
.volume-slider trough highlight { background: $yellow; }
.volume-mute.active { background: rgba(249, 226, 175, 0.12); color: $yellow; }
.volume-sink.active { background: rgba(249, 226, 175, 0.10); color: $yellow; }
.volume-sink-dot { color: $overlay1; margin-right: 8px; }
.volume-sink.active .volume-sink-dot { color: $yellow; }
.volume-sink-name { color: $text; }
```

- [ ] **Step 5: Build and verify**

Run: `just fmt && NIXHOST=laptop just build && systemctl --user restart eww-bar`
Manual:
- Left-click the volume module → popup opens near the volume icon.
- Drag the slider → system volume changes and the bar `%` follows.
- Click "Mute" → toggles; label flips to "Unmute".
- Click a non-active sink row → audio output switches; the active dot moves.
- Click outside / the volume icon again → closes.

- [ ] **Step 6: Commit**

```bash
git add modules/desktop/home/hyprland/eww/eww.yuck modules/desktop/home/hyprland/eww/eww.scss
git commit -m "feat(eww): add volume popup with slider, mute, and sink switch

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Bluetooth backend (devices + powered + control)

**Files:**
- Modify: `<eww>/scripts/eww_bar_backend/collectors.py` (`bluetooth_state_from_text` → add `powered` + `devices`)
- Modify: `<eww>/scripts/eww_bar_backend/control.py` (add `toggle_bluetooth_power`, `disconnect_bluetooth`; command + arg parsing + import)
- Modify: `<eww>/scripts/eww_bar_backend/common.py` (bluetooth default with new fields)
- Modify: `<eww>/scripts/eww_bar_backend/state.py` (seed) + `<eww>/eww.yuck` (deflisten seed)
- Test: `tests/eww_bar_backend/test_bluetooth.py`

**Interfaces:**
- Produces: `bluetooth_state()` dict now includes `powered: "true"/"false"` and `devices: [{mac, name, battery, connected}]`; control verbs `bluetooth power-toggle | disconnect <mac>`. Task 6 consumes these.

- [ ] **Step 1: Write the failing test**

Create `tests/eww_bar_backend/test_bluetooth.py`:

```python
import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, control  # noqa: E402


class BluetoothStateTests(unittest.TestCase):
    def test_powered_and_devices_parsed(self):
        controller = "Controller AA:BB Alias: MyBT\n\tPowered: yes\n\tAlias: MyBT\n"
        devices = "Device 80:99:E7 WF-1000XM5\n"
        info = {"80:99:E7": "\tAlias: WF-1000XM5\n\tBattery Percentage: 0x50 (80)\n"}
        state = collectors.bluetooth_state_from_text(controller, devices, info)
        self.assertEqual(state["powered"], "true")
        self.assertEqual(len(state["devices"]), 1)
        self.assertEqual(state["devices"][0]["mac"], "80:99:E7")
        self.assertEqual(state["devices"][0]["name"], "WF-1000XM5")
        self.assertEqual(state["devices"][0]["battery"], "80%")
        self.assertEqual(state["devices"][0]["connected"], "true")

    def test_powered_off_no_devices(self):
        controller = "Controller AA:BB\n\tPowered: no\n"
        state = collectors.bluetooth_state_from_text(controller, "", {})
        self.assertEqual(state["powered"], "false")
        self.assertEqual(state["devices"], [])


class BluetoothControlTests(unittest.TestCase):
    def _capture(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        return calls, fake_run

    def test_power_toggle_turns_off_when_on(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "bluetooth_state", side_effect=[{"powered": "true"}, {}]):
                control.toggle_bluetooth_power()
        self.assertEqual(calls, [["bluetoothctl", "power", "off"]])

    def test_power_toggle_turns_on_when_off(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "bluetooth_state", side_effect=[{"powered": "false"}, {}]):
                control.toggle_bluetooth_power()
        self.assertEqual(calls, [["bluetoothctl", "power", "on"]])

    def test_disconnect(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "bluetooth_state", return_value={}):
                control.disconnect_bluetooth("80:99:E7")
        self.assertEqual(calls, [["bluetoothctl", "disconnect", "80:99:E7"]])

    def test_arg_parsing(self):
        self.assertEqual(
            control.control_payload_from_args(["bluetooth", "power-toggle"]),
            {"command": "bluetooth", "action": "power-toggle"},
        )
        self.assertEqual(
            control.control_payload_from_args(["bluetooth", "disconnect", "80:99:E7"]),
            {"command": "bluetooth", "action": "disconnect", "mac": "80:99:E7"},
        )


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 tests/eww_bar_backend/test_bluetooth.py -v`
Expected: FAIL — `KeyError: 'powered'` / `control.toggle_bluetooth_power` missing.

- [ ] **Step 3: Add `powered` + `devices` to the collector**

In `<eww>/scripts/eww_bar_backend/collectors.py`, in `bluetooth_state_from_text`, build a structured device list and add the two fields to **both** return statements. Replace the function body's device-loop and returns so it reads:

```python
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
            {"mac": address, "name": alias, "battery": battery, "connected": "true"}
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
```

- [ ] **Step 4: Add control verbs**

In `<eww>/scripts/eww_bar_backend/control.py`, add `bluetooth_state` to the `from .collectors import (…)` line (currently `from .collectors import media_state, refresh_ai_usage, volume_state`):

```python
from .collectors import bluetooth_state, media_state, refresh_ai_usage, volume_state
```

Add these functions (near `control_media`):

```python
def toggle_bluetooth_power():
    current = bluetooth_state().get("powered", "false")
    target = "off" if current == "true" else "on"
    subprocess.run(
        ["bluetoothctl", "power", target],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return bluetooth_state()


def disconnect_bluetooth(mac):
    if not mac:
        raise ValueError("bluetooth disconnect requires a device address")
    subprocess.run(
        ["bluetoothctl", "disconnect", mac],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return bluetooth_state()
```

Add a command branch in `handle_control_command` (after the `media` branch):

```python
    if command == "bluetooth":
        action = payload.get("action", "")
        if action == "power-toggle":
            value = toggle_bluetooth_power()
        elif action == "disconnect":
            value = disconnect_bluetooth(payload.get("mac", ""))
        else:
            raise ValueError("bluetooth action must be power-toggle or disconnect")
        state.update(bluetooth=value)
        return {"ok": True, "command": "bluetooth", "action": action, "bluetooth": value}
```

Add arg parsing in `control_payload_from_args` (after the `media` line):

```python
    if args[0] == "bluetooth" and len(args) >= 2:
        payload = {"command": "bluetooth", "action": args[1]}
        if args[1] == "disconnect" and len(args) >= 3:
            payload["mac"] = args[2]
        return payload
```

Update `CONTROL_USAGE` to append `| bluetooth power-toggle|disconnect <mac>` before the closing quote.

- [ ] **Step 5: Seed the new shape**

In `<eww>/scripts/eww_bar_backend/common.py`, add:

```python
BLUETOOTH_DEFAULT = {"text": "", "tooltip": "", "class": "", "powered": "false", "devices": []}
```

In `<eww>/scripts/eww_bar_backend/state.py`, import `BLUETOOTH_DEFAULT` and change `"bluetooth": EMPTY_MODULE.copy()` to `"bluetooth": BLUETOOTH_DEFAULT.copy()`.

In `<eww>/eww.yuck` line 1, replace `"bluetooth":{"text":"","tooltip":"","class":""}` with:

```json
"bluetooth":{"text":"","tooltip":"","class":"","powered":"false","devices":[]}
```

- [ ] **Step 6: Run tests + compile + eval**

Run: `python3 tests/eww_bar_backend/test_bluetooth.py -v`
Expected: PASS.
Run: `python3 -m py_compile modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/*.py`
Expected: no output.

- [ ] **Step 7: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/control.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/common.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/state.py \
        modules/desktop/home/hyprland/eww/eww.yuck \
        tests/eww_bar_backend/test_bluetooth.py
git commit -m "feat(eww): add structured bluetooth devices and power/disconnect control

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Bluetooth popup (widget + styling + bar wiring)

**Files:**
- Modify: `<eww>/eww.yuck` (`bluetooth_popup` window + `bluetooth_panel`; bluetooth module → button)
- Modify: `<eww>/eww.scss` (bluetooth specifics)

**Interfaces:**
- Consumes: shared `popup_header`/`.popup*` (Task 4); `bar_state.bluetooth.{powered,devices}` and `bluetooth power-toggle|disconnect` (Task 5); `eww-popup` (Task 1).

- [ ] **Step 1: Add the widgets**

In `<eww>/eww.yuck`:

```lisp
(defwidget bluetooth_panel []
  (box :class "popup accent-bluetooth" :orientation "v" :space-evenly false
    (popup_header :icon "󰂯" :title "Bluetooth"
      :subtitle {bar_state.bluetooth.powered == "true" ? "Powered on" : "Powered off"})
    (box :class "popup-body" :orientation "v" :space-evenly false
      (button :class "popup-action bluetooth-power ${bar_state.bluetooth.powered == 'true' ? 'active' : ''}"
        :onclick "eww-barctl --quiet bluetooth power-toggle"
        (box :orientation "h" :space-evenly false
          (label :class "popup-action-icon" :text {bar_state.bluetooth.powered == "true" ? "󰂯" : "󰂲"})
          (label :class "popup-action-label" :halign "start"
            :text {bar_state.bluetooth.powered == "true" ? "Turn off" : "Turn on"})))
      (box :class "popup-section-label" :orientation "h" :space-evenly false
        (label :halign "start" :text "CONNECTED"))
      (box :class "bluetooth-devices" :orientation "v" :space-evenly false
        (box :class "popup-empty" :orientation "h" :space-evenly false
          :visible {arraylength(bar_state.bluetooth.devices) == 0}
          (label :halign "start" :text "No connected devices"))
        (for device in {bar_state.bluetooth.devices}
          (box :class "popup-row bluetooth-device" :orientation "h" :space-evenly false
            (label :class "bluetooth-device-name" :halign "start" :text {device.name})
            (label :class "bluetooth-device-battery" :visible {device.battery != ""} :text {device.battery})
            (button :class "bluetooth-disconnect" :halign "end" :hexpand true
              :onclick "eww-barctl --quiet bluetooth disconnect ${device.mac}"
              (label :text "󰅖")))))
      (button :class "popup-footer-action"
        :onclick "hyprctl dispatch exec blueman-manager"
        (label :text "Open blueman")))))

(defwindow bluetooth_popup
  :monitor 0
  :geometry (geometry :x "205px" :y "50px" :width "300px" :height "230px" :anchor "top right")
  :stacking "fg"
  :exclusive false
  :focusable false
  :namespace "eww-bluetooth"
  (bluetooth_panel))
```

- [ ] **Step 2: Convert the bluetooth module to a button**

In `<eww>/eww.yuck` (tray-island), replace the bluetooth `label`:

```lisp
          (label :class "module bluetooth ${bar_state.bluetooth.class}" :visible {bar_state.bluetooth.text != ""} :tooltip {bar_state.bluetooth.tooltip} :text {bar_state.bluetooth.text}))
```

with a button wrapping the same label:

```lisp
          (button :class "module-button bluetooth ${bar_state.bluetooth.class}"
            :visible {bar_state.bluetooth.text != ""}
            :onclick "eww-popup toggle bluetooth_popup ${output}"
            (label :tooltip {bar_state.bluetooth.tooltip} :text {bar_state.bluetooth.text})))
```

- [ ] **Step 3: Add bluetooth SCSS**

In `<eww>/eww.scss`, add:

```scss
/* Bluetooth popup specifics */
.bluetooth-power.active { background: rgba(116, 199, 236, 0.12); color: $sapphire; }
.bluetooth-device-name { color: $text; font-weight: 700; }
.bluetooth-device-battery { color: $overlay1; font-size: 10px; margin-left: 8px; }
.bluetooth-disconnect { border-radius: 6px; color: $overlay1; min-height: 24px; min-width: 26px; }
.bluetooth-disconnect:hover { background: rgba(243, 139, 168, 0.14); color: $red; }
```

- [ ] **Step 4: Build and verify**

Run: `just fmt && NIXHOST=laptop just build && systemctl --user restart eww-bar`
Manual: click bluetooth module → popup opens; power toggle flips the controller; a connected device shows name+battery and the disconnect button drops it; "Open blueman" launches blueman-manager; outside-click closes.

- [ ] **Step 5: Commit**

```bash
git add modules/desktop/home/hyprland/eww/eww.yuck modules/desktop/home/hyprland/eww/eww.scss
git commit -m "feat(eww): add bluetooth popup with power toggle and device list

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Network backend (wifi flag + toggle)

**Files:**
- Modify: `<eww>/scripts/eww_bar_backend/collectors.py` (rename body to `network_connection_state`; add `network_radio_enabled` + thin `network_state` wrapper)
- Modify: `<eww>/scripts/eww_bar_backend/control.py` (add `toggle_wifi`; command + parsing + import)
- Modify: `<eww>/scripts/eww_bar_backend/common.py` + `state.py` + `<eww>/eww.yuck` (seed `wifi_enabled`)
- Test: `tests/eww_bar_backend/test_network.py`

**Interfaces:**
- Produces: `network_state()` dict gains `wifi_enabled: "true"/"false"`; `network_radio_enabled() -> "true"/"false"`; control verb `network wifi-toggle`. Task 8 consumes `bar_state.network.{text,tooltip,class,wifi_enabled}`.

- [ ] **Step 1: Write the failing test**

Create `tests/eww_bar_backend/test_network.py`:

```python
import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, control  # noqa: E402


class NetworkRadioTests(unittest.TestCase):
    def test_enabled(self):
        with unittest.mock.patch.object(collectors, "run_text", return_value="enabled\n"):
            self.assertEqual(collectors.network_radio_enabled(), "true")

    def test_disabled(self):
        with unittest.mock.patch.object(collectors, "run_text", return_value="disabled\n"):
            self.assertEqual(collectors.network_radio_enabled(), "false")

    def test_state_includes_wifi_enabled(self):
        with unittest.mock.patch.object(
            collectors, "network_connection_state", return_value={"text": "x", "class": "wifi"}
        ):
            with unittest.mock.patch.object(collectors, "network_radio_enabled", return_value="true"):
                state = collectors.network_state()
        self.assertEqual(state["wifi_enabled"], "true")
        self.assertEqual(state["class"], "wifi")


class WifiToggleTests(unittest.TestCase):
    def _capture(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        return calls, fake_run

    def test_toggle_off_when_on(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "network_radio_enabled", return_value="true"):
                with unittest.mock.patch.object(control, "network_state", return_value={}):
                    control.toggle_wifi()
        self.assertEqual(calls, [["nmcli", "radio", "wifi", "off"]])

    def test_toggle_on_when_off(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "network_radio_enabled", return_value="false"):
                with unittest.mock.patch.object(control, "network_state", return_value={}):
                    control.toggle_wifi()
        self.assertEqual(calls, [["nmcli", "radio", "wifi", "on"]])

    def test_arg_parsing(self):
        self.assertEqual(
            control.control_payload_from_args(["network", "wifi-toggle"]),
            {"command": "network", "action": "wifi-toggle"},
        )


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 tests/eww_bar_backend/test_network.py -v`
Expected: FAIL — `network_radio_enabled` / `network_connection_state` / `toggle_wifi` missing.

- [ ] **Step 3: Add the radio flag + wrapper**

In `<eww>/scripts/eww_bar_backend/collectors.py`, rename the existing `def network_state():` to `def network_connection_state():` (body unchanged). Then add directly below it:

```python
def network_radio_enabled():
    return "true" if run_text(["nmcli", "radio", "wifi"]).strip() == "enabled" else "false"


def network_state():
    result = network_connection_state()
    result["wifi_enabled"] = network_radio_enabled()
    return result
```

- [ ] **Step 4: Add the control verb**

In `<eww>/scripts/eww_bar_backend/control.py`, add `network_radio_enabled, network_state` to the collectors import line. Add:

```python
def toggle_wifi():
    target = "off" if network_radio_enabled() == "true" else "on"
    subprocess.run(
        ["nmcli", "radio", "wifi", target],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    return network_state()
```

In `handle_control_command`:

```python
    if command == "network":
        action = payload.get("action", "")
        if action == "wifi-toggle":
            value = toggle_wifi()
        else:
            raise ValueError("network action must be wifi-toggle")
        state.update(network=value)
        return {"ok": True, "command": "network", "action": action, "network": value}
```

In `control_payload_from_args`:

```python
    if args[0] == "network" and len(args) >= 2:
        return {"command": "network", "action": args[1]}
```

Append `| network wifi-toggle` to `CONTROL_USAGE`.

- [ ] **Step 5: Seed the new field**

In `common.py` add:

```python
NETWORK_DEFAULT = {"text": "", "tooltip": "", "class": "", "wifi_enabled": "false"}
```

In `state.py` import `NETWORK_DEFAULT` and change `"network": EMPTY_MODULE.copy()` to `"network": NETWORK_DEFAULT.copy()`.

In `<eww>/eww.yuck` line 1, replace `"network":{"text":"","tooltip":"","class":""}` with:

```json
"network":{"text":"","tooltip":"","class":"","wifi_enabled":"false"}
```

- [ ] **Step 6: Run tests + compile**

Run: `python3 tests/eww_bar_backend/test_network.py -v`
Expected: PASS.
Run: `python3 -m py_compile modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/*.py`
Expected: no output.

- [ ] **Step 7: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/control.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/common.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/state.py \
        modules/desktop/home/hyprland/eww/eww.yuck \
        tests/eww_bar_backend/test_network.py
git commit -m "feat(eww): expose wifi radio state and wifi-toggle control

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Network popup (widget + styling + bar wiring)

**Files:**
- Modify: `<eww>/eww.yuck` (`network_popup` + `network_panel`; network module → button)
- Modify: `<eww>/eww.scss` (network specifics)

**Interfaces:**
- Consumes: shared `popup_header`/`.popup*` (Task 4); `bar_state.network.{text,tooltip,class,wifi_enabled}` + `network wifi-toggle` (Task 7); `eww-popup` (Task 1).

- [ ] **Step 1: Add the widgets**

```lisp
(defwidget network_panel []
  (box :class "popup accent-network" :orientation "v" :space-evenly false
    (popup_header :icon "" :title "Network"
      :subtitle {bar_state.network.class == "disconnected" ? "Disconnected" : "Connected"})
    (box :class "popup-body" :orientation "v" :space-evenly false
      (box :class "network-card" :orientation "v" :space-evenly false
        (label :class "network-primary" :halign "start"
          :text {bar_state.network.text != "" ? bar_state.network.text : "No connection"})
        (label :class "network-detail" :halign "start" :text {bar_state.network.tooltip}))
      (button :class "popup-action network-wifi ${bar_state.network.wifi_enabled == 'true' ? 'active' : ''}"
        :onclick "eww-barctl --quiet network wifi-toggle"
        (box :orientation "h" :space-evenly false
          (label :class "popup-action-icon" :text {bar_state.network.wifi_enabled == "true" ? "" : "󰖪"})
          (label :class "popup-action-label" :halign "start"
            :text {bar_state.network.wifi_enabled == "true" ? "Wi-Fi on" : "Wi-Fi off"})))
      (button :class "popup-footer-action"
        :onclick "hyprctl dispatch exec 'kitty -- nmtui'"
        (label :text "Advanced…")))))

(defwindow network_popup
  :monitor 0
  :geometry (geometry :x "275px" :y "50px" :width "300px" :height "200px" :anchor "top right")
  :stacking "fg"
  :exclusive false
  :focusable false
  :namespace "eww-network"
  (network_panel))
```

- [ ] **Step 2: Convert the network module to a button**

In `<eww>/eww.yuck` (tray-island), replace the network `label`:

```lisp
          (label :class "module network ${bar_state.network.class}" :tooltip {bar_state.network.tooltip} :text {bar_state.network.text})
```

with:

```lisp
          (button :class "module-button network ${bar_state.network.class}"
            :onclick "eww-popup toggle network_popup ${output}"
            (label :tooltip {bar_state.network.tooltip} :text {bar_state.network.text}))
```

- [ ] **Step 3: Add network SCSS**

```scss
/* Network popup specifics */
.network-card { background: rgba(30, 30, 46, 0.74); border-radius: 8px; padding: 10px 12px; }
.network-primary { color: $text; font-size: 14px; font-weight: 800; }
.network-detail { color: $overlay1; font-size: 10px; margin-top: 2px; }
.network-wifi.active { background: rgba(250, 179, 135, 0.12); color: $peach; }
```

- [ ] **Step 4: Build and verify**

Run: `just fmt && NIXHOST=laptop just build && systemctl --user restart eww-bar`
Manual: click network module → popup shows current connection + IP; Wi-Fi toggle flips `nmcli radio wifi` (verify with `nmcli radio wifi`); "Advanced…" opens `kitty -- nmtui`; outside-click closes.

- [ ] **Step 5: Commit**

```bash
git add modules/desktop/home/hyprland/eww/eww.yuck modules/desktop/home/hyprland/eww/eww.scss
git commit -m "feat(eww): add network popup with wifi toggle and nmtui launch

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 9: Battery backend (status/health/power/time)

**Files:**
- Modify: `<eww>/scripts/eww_bar_backend/collectors.py` (`battery_state` → add fields; accept `root` param)
- Modify: `<eww>/scripts/eww_bar_backend/common.py` (extend `BATTERY_DEFAULT`) + `<eww>/eww.yuck` (deflisten seed)
- Test: `tests/eww_bar_backend/test_battery.py`

**Interfaces:**
- Produces: `battery_state()` dict gains `status`, `time`, `health` (`"NN%"`/`"--"`), `power` (`"N.N W"`/`"—"`). Task 10 consumes `bar_state.battery.{capacity,class,status,time,health,power}`.

- [ ] **Step 1: Write the failing test**

Create `tests/eww_bar_backend/test_battery.py`:

```python
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors  # noqa: E402


def _write_bat(root, **files):
    bat = Path(root) / "BAT0"
    bat.mkdir(parents=True)
    for name, value in files.items():
        (bat / name).write_text(str(value))


class BatteryStateTests(unittest.TestCase):
    def test_charge_units_health_and_power(self):
        with TemporaryDirectory() as root:
            _write_bat(
                root,
                capacity="80",
                status="Discharging",
                charge_now="2400000",
                current_now="1000000",
                charge_full="3000000",
                charge_full_design="3077000",
                voltage_now="12000000",
            )
            state = collectors.battery_state(root=Path(root))
        self.assertEqual(state["status"], "Discharging")
        self.assertEqual(state["health"], "97%")          # 3000000/3077000
        self.assertEqual(state["power"], "12.0 W")          # 1.0 A * 12.0 V
        self.assertEqual(state["capacity"], 80)
        self.assertNotEqual(state["time"], "")

    def test_full_reports_zero_draw(self):
        with TemporaryDirectory() as root:
            _write_bat(
                root,
                capacity="100",
                status="Full",
                charge_now="3077000",
                current_now="0",
                charge_full="3077000",
                charge_full_design="3077000",
                voltage_now="12716000",
            )
            state = collectors.battery_state(root=Path(root))
        self.assertEqual(state["status"], "Full")
        self.assertEqual(state["health"], "100%")
        self.assertEqual(state["power"], "—")


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 tests/eww_bar_backend/test_battery.py -v`
Expected: FAIL — `battery_state() got an unexpected keyword argument 'root'` / `KeyError: 'health'`.

- [ ] **Step 3: Extend the battery collector**

In `<eww>/scripts/eww_bar_backend/collectors.py`, replace `def battery_state():` with a version that takes `root`, reads the design capacity + voltage, and returns the new fields:

```python
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
```

> Note: copy the battery icon glyphs (`󰂄 󰁻 󰁼 󰁾 󰂀 󰁹`) verbatim from the previous `battery_state` body.

- [ ] **Step 4: Extend the default + seed**

In `common.py`, replace `BATTERY_DEFAULT` with:

```python
BATTERY_DEFAULT = {
    "text": "󰂄",
    "alt": "󰂄",
    "capacity": 100,
    "class": "charging",
    "status": "Unknown",
    "time": "N/A",
    "health": "--",
    "power": "—",
}
```

In `<eww>/eww.yuck` line 1, replace `"battery":{"text":"","alt":"","capacity":100,"class":""}` with:

```json
"battery":{"text":"","alt":"","capacity":100,"class":"","status":"Unknown","time":"N/A","health":"--","power":"—"}
```

- [ ] **Step 5: Run tests + compile**

Run: `python3 tests/eww_bar_backend/test_battery.py -v`
Expected: PASS.
Run: `python3 -m py_compile modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/*.py`
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py \
        modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/common.py \
        modules/desktop/home/hyprland/eww/eww.yuck \
        tests/eww_bar_backend/test_battery.py
git commit -m "feat(eww): add battery status/health/power/time to state

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 10: Battery popup (widget + styling + bar wiring)

**Files:**
- Modify: `<eww>/eww.yuck` (`battery_popup` + `battery_panel` + `battery_row`; battery module onclick)
- Modify: `<eww>/eww.scss` (battery specifics)

**Interfaces:**
- Consumes: shared `popup_header`/`.popup*` (Task 4); `bar_state.battery.{capacity,class,status,time,health,power}` (Task 9); `eww-popup` (Task 1).

- [ ] **Step 1: Add the widgets**

```lisp
(defwidget battery_row [label value]
  (box :class "popup-row battery-row" :orientation "h" :space-evenly false
    (label :class "battery-row-label" :halign "start" :text label)
    (label :class "battery-row-value" :halign "end" :hexpand true :text value)))

(defwidget battery_panel []
  (box :class "popup accent-battery" :orientation "v" :space-evenly false
    (popup_header :icon "󰁹" :title "Battery" :subtitle {bar_state.battery.status})
    (box :class "popup-body" :orientation "v" :space-evenly false
      (box :class "battery-headline" :orientation "h" :space-evenly false
        (label :class "battery-capacity" :text "${bar_state.battery.capacity}%")
        (label :class "battery-status" :halign "end" :hexpand true :text {bar_state.battery.status}))
      (progress :class "popup-meter battery-meter ${bar_state.battery.class}" :value {bar_state.battery.capacity})
      (box :class "battery-rows" :orientation "v" :space-evenly false
        (battery_row :label "Time" :value {bar_state.battery.time})
        (battery_row :label "Health" :value {bar_state.battery.health})
        (battery_row :label "Power draw" :value {bar_state.battery.power})))))

(defwindow battery_popup
  :monitor 0
  :geometry (geometry :x "8px" :y "50px" :width "280px" :height "220px" :anchor "top right")
  :stacking "fg"
  :exclusive false
  :focusable false
  :namespace "eww-battery"
  (battery_panel))
```

- [ ] **Step 2: Wire the battery module's click**

In `<eww>/eww.yuck` (session-island), replace the battery button's `:onclick` (currently `eww update show_battery_time=…`) with:

```lisp
            :onclick "eww-popup toggle battery_popup ${output}"
```

Leave the label (`{show_battery_time == "true" ? bar_state.battery.alt : bar_state.battery.text}`) as-is — the bar still shows charge %.

- [ ] **Step 3: Add battery SCSS**

```scss
/* Battery popup specifics */
.battery-capacity { color: $text; font-size: 22px; font-weight: 900; }
.battery-status { color: $overlay1; font-size: 11px; }
.battery-meter { margin: 8px 0 4px; }
.battery-meter trough progress { background: $blue; }
.battery-meter.critical trough progress { background: $red; }
.battery-meter.warning trough progress { background: $yellow; }
.battery-meter.charging trough progress { background: $green; }
.battery-row { min-height: 26px; }
.battery-row-label { color: $overlay1; font-size: 11px; }
.battery-row-value { color: $text; font-size: 11px; font-weight: 800; }
```

- [ ] **Step 4: Build and verify**

Run: `just fmt && NIXHOST=laptop just build && systemctl --user restart eww-bar`
Manual: click battery module → popup shows capacity, status, a colored meter, and Time/Health/Power rows with real values; outside-click closes.

- [ ] **Step 5: Commit**

```bash
git add modules/desktop/home/hyprland/eww/eww.yuck modules/desktop/home/hyprland/eww/eww.scss
git commit -m "feat(eww): add battery detail popup

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 11: Offset/size calibration + full verification

**Files:**
- Modify: `<eww>/eww.yuck` (tune each popup window's `:x` and `:height`)

**Interfaces:**
- Consumes: all popups (Tasks 2/4/6/8/10).

- [ ] **Step 1: Capture a calibration screenshot per popup**

Build is already current. For each module, open its popup and screenshot:

```bash
eww-popup toggle volume_popup eDP-1 ; sleep 0.3 ; grim /tmp/eww-volume.png
eww-popup close
```

Repeat for `bluetooth_popup`, `network_popup`, `battery_popup` (and the retrofitted `ai_usage_popup`). Read each PNG.

- [ ] **Step 2: Adjust offsets and heights**

For each `defwindow …_popup` in `<eww>/eww.yuck`, adjust `:x` so the popup sits under its module's icon (increase `:x` to move the popup left, away from the right edge), and adjust `:height` so the card's bottom edge matches its content (no clipped rows, no large transparent gap below the card — the gap is a dead zone that won't close on click). Starting values: battery `8px`, volume `70px`, bluetooth `205px`, network `275px`.

If any popup's content can exceed its height (e.g. many sinks/devices), wrap the variable list (`volume-sinks` / `bluetooth-devices`) in a `(scroll :vscroll true :hscroll false :vexpand true …)` and keep the window height fixed — mirror the `ai-usage-scroll` pattern already in `eww.yuck`.

- [ ] **Step 3: Rebuild and re-verify positions**

Run: `NIXHOST=laptop just build && systemctl --user restart eww-bar`
Re-screenshot; confirm each popup is under its module on **both** `eDP-1` and `HDMI-A-1` (offsets should match on both since they are measured from the right edge).

- [ ] **Step 4: Run the full backend suite + eval + format**

Run: `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py' -v`
Expected: all tests PASS (existing + `test_popups`, `test_volume`, `test_bluetooth`, `test_network`, `test_battery`).

Run: `cd /home/lemonilemon/nixos-config && just fmt && NIXHOST=laptop just test`
Expected: format clean; eval prints a `.drvPath`.

- [ ] **Step 5: Full manual acceptance pass**

Confirm the entire spec's manual checklist:
- Each of volume/bluetooth/network/battery opens under its module, on both monitors.
- Same-module re-click, different-module click, and empty-desktop click all close/switch correctly (one click each).
- Volume: slider/mute/sink all work. Bluetooth: power/disconnect/blueman. Network: wifi-toggle/nmtui. Battery: detail values populate.
- Retrofitted AI-usage + display-mode popups still open and close on outside click.

- [ ] **Step 6: Commit**

```bash
git add modules/desktop/home/hyprland/eww/eww.yuck
git commit -m "feat(eww): calibrate module popup offsets and sizes

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Notes for the implementer

- The `for` loop over `bar_state.volume.sinks` / `bar_state.bluetooth.devices` is safe here (these popups have no tab-switching, unlike the AI-usage quota cards whose comment warns about eww re-appending `for` children). If a list ever renders blank after an update, fall back to a small fixed set of indexed rows (`bar_state.…[0]`, `[1]`, …) as the quota cards do.
- eww `scale` fill/knob selectors (`trough highlight`, `trough slider`) can vary; if the slider fill isn't colored, try `trough progress` as an alternative selector. Functionality (`:onchange`) is independent of styling.
- Watch for a `scale` feedback loop: the volume watcher rewrites `bar_state.volume.percent` on every PipeWire event, and updating a `scale`'s `:value` must not re-fire `:onchange` (eww 0.6 does not, in practice). If you observe the volume "fighting" the slider while dragging, debounce by only re-reading the percent when `volume_popup` is closed, or drop the `:value` binding to a separate var updated on open.
- All state-shape edits must keep the deflisten seed (eww.yuck line 1), `common.py`, and `state.py` in sync, or the bar renders the literal default until the first collector runs.
