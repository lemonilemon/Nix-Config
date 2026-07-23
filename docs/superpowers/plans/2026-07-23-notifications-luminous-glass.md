# Notifications + Luminous-Glass Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep dunst as the notification daemon but make notifications first-class (eww notification center grouped by app, bar bell with DND, Wayland keybinds), and restyle bar/popups/toasts in the "luminous glass" language with the palette extracted to one file.

**Architecture:** A new `notifications.py` backend module (same pattern as `display.py`) parses `dunstctl history` JSON into grouped state, owns ephemeral UI state (collapsed groups, last-seen), and exposes control verbs over the existing socket. The eww side adds one popup window registered in the existing popup system plus a bell module. Styling is centralized: `theme/palette.nix` → generated `_palette.scss` + dunst colors; two SCSS mixins define the glass surfaces.

**Tech Stack:** eww 0.6.0 (yuck + SCSS via grass), Python 3 stdlib only, dunst 1.13.2 (`dunstctl`), Home Manager + NixOS flake, Hyprland layerrules/binds.

**Spec:** `docs/superpowers/specs/2026-07-23-notifications-luminous-glass-design.md`

## Global Constraints

- Implementers must NOT run `sudo`, `just build`, `nixos-rebuild`, or `systemctl`. The user triggers rebuilds; the controller does live verification (Task 8).
- Python: stdlib only, no third-party imports.
- Test suite: `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py'` — baseline 48 tests, must stay green; every task adds its own.
- After editing any Python file: `python3 -m py_compile <file>` must pass.
- Nerd-font glyphs in yuck/py must be pasted exactly; after editing, verify with `python3 -c "print(open('<file>').read().count('\N{BELL}') >= 0)"`-style spot checks listed per task (a past task silently lost glyphs).
- eww window/verb names used across tasks are load-bearing: window `notif_center_popup`, state key `notifications`, verb group `notif`.
- Booleans inside `bar_state` are strings (`"true"`/`"false"`), matching the existing repo convention (the spec's JSON sketch used raw bools; strings win).
- All external calls go through `run_text` (2s timeout) or the DEVNULL/`check=False` subprocess pattern already used in `control.py`.
- Commit after every task; nixfmt runs from the pre-commit hook.

---

### Task 1: Palette source of truth (`palette.nix` → `_palette.scss` + dunst)

**Files:**
- Create: `modules/desktop/home/hyprland/theme/palette.nix`
- Modify: `modules/desktop/home/hyprland/eww/default.nix` (add generated `_palette.scss`)
- Modify: `modules/desktop/home/hyprland/eww/eww.scss:1-21` (replace variable block with import)
- Modify: `modules/desktop/home/hyprland/dunst.nix` (reference palette instead of hex literals)

**Interfaces:**
- Produces: `import ../theme/palette.nix` (from `eww/`) / `import ./theme/palette.nix` (from `hyprland/`) → attrset of 21 hex strings keyed `rosewater pink mauve red maroon peach yellow green teal sapphire blue lavender text subtext0 overlay1 overlay0 surface1 surface0 mantle base crust`; SCSS vars `$mauve` etc. available via `@import "palette";`. This task changes **zero rendered colors** — pure extraction.

- [ ] **Step 1: Create `theme/palette.nix`**

```nix
# Catppuccin Mocha — single source of truth for eww (_palette.scss), dunst,
# and any future consumer. Keys match the SCSS variable names.
{
  rosewater = "#f5e0dc";
  pink = "#f5c2e7";
  mauve = "#cba6f7";
  red = "#f38ba8";
  maroon = "#eba0ac";
  peach = "#fab387";
  yellow = "#f9e2af";
  green = "#a6e3a1";
  teal = "#94e2d5";
  sapphire = "#74c7ec";
  blue = "#89b4fa";
  lavender = "#b4befe";
  text = "#cdd6f4";
  subtext0 = "#a6adc8";
  overlay1 = "#7f849c";
  overlay0 = "#6c7086";
  surface1 = "#45475a";
  surface0 = "#313244";
  mantle = "#181825";
  base = "#1e1e2e";
  crust = "#11111b";
}
```

- [ ] **Step 2: Verify the attrset evaluates**

Run: `nix eval --impure --expr '(import ./modules/desktop/home/hyprland/theme/palette.nix).mauve'`
Expected: `"#cba6f7"`

- [ ] **Step 3: Generate `_palette.scss` in `eww/default.nix`**

In the `let` block (after `cfg = ...;`), add:

```nix
  palette = import ../theme/palette.nix;
  paletteScss = pkgs.writeText "_palette.scss" (
    lib.concatStringsSep "\n" (lib.mapAttrsToList (name: value: "$${name}: ${value};") palette)
  );
```

In the config body, next to the existing `xdg.configFile."eww/eww.scss"` line, add:

```nix
    xdg.configFile."eww/_palette.scss".source = paletteScss;
```

Note the escaping: inside a Nix string, a literal `$` followed by `{` must be written `$$` → the template `"$${name}: ${value};"` renders `$mauve: #cba6f7;`.

- [ ] **Step 4: Replace the variable block in `eww.scss`**

Delete lines 1–21 (`$rosewater: #f5e0dc;` … `$crust: #11111b;`) and replace with:

```scss
@import "palette";
```

(grass resolves `@import "palette"` to `_palette.scss` in the same directory.)

- [ ] **Step 5: Point `dunst.nix` at the palette**

At the top of the module add `palette = import ./theme/palette.nix;` to a `let` block:

```nix
{
  lib,
  config,
  ...
}:
let
  palette = import ./theme/palette.nix;
in
```

Replace the three urgency sections' hex literals with palette references **producing byte-identical values**:

```nix
        urgency_low = {
          background = palette.base;
          foreground = palette.subtext0;
          frame_color = palette.surface0;
          timeout = 4;
        };

        urgency_normal = {
          background = palette.base;
          foreground = palette.text;
          frame_color = palette.mauve;
          timeout = 6;
        };

        urgency_critical = {
          background = palette.base;
          foreground = palette.red;
          frame_color = palette.red;
          timeout = 0; # never auto-dismiss
        };
```

- [ ] **Step 6: Sanity-check generated SCSS text**

Run: `nix eval --impure --expr 'let lib = (import <nixpkgs> {}).lib; p = import ./modules/desktop/home/hyprland/theme/palette.nix; in lib.concatStringsSep "\n" (lib.mapAttrsToList (n: v: "$${n}: ${v};") p)' | head -c 200`
Expected: output contains `$base: #1e1e2e;` and `$mauve: #cba6f7;` (attr order is alphabetical — fine, SCSS vars are order-independent among themselves and the import precedes all uses).

- [ ] **Step 7: Commit**

```bash
git add modules/desktop/home/hyprland/theme/palette.nix modules/desktop/home/hyprland/eww/default.nix modules/desktop/home/hyprland/eww/eww.scss modules/desktop/home/hyprland/dunst.nix
git commit -m "refactor(theme): extract Catppuccin palette to palette.nix"
```

---

### Task 2: Luminous-glass SCSS restyle + blur layerrules

**Files:**
- Modify: `modules/desktop/home/hyprland/eww/eww.scss`
- Modify: `modules/desktop/home/hyprland/default.nix:136-142` (layerrule block)

**Interfaces:**
- Consumes: `@import "palette";` from Task 1.
- Produces: SCSS mixins `glass-island($accent)` and `glass-popup($accent)` used by Task 6 (notif center) and by the wallpaper plan. Accent mapping (fixed): workspace/clock → `$green`, media → `$teal`, window → `$lavender`, tray → `$overlay1`, sys → `$mauve`, session → `$blue`; popups: volume `$yellow`, bluetooth `$sapphire`, network `$peach`, battery `$blue`, ai-usage `$sapphire`, display-mode `$yellow`, notif center `$mauve` (class `accent-notif`, added in Task 6).

- [ ] **Step 1: Add the glass mixins right after the `@import`**

```scss
// Luminous glass: translucent surface + accent-tinted hairline + faint glow.
// Alphas/radii are calibration values — tune in the final live pass only.
@mixin glass-island($accent) {
  background: rgba(17, 17, 27, 0.60);
  border: 1px solid rgba($accent, 0.35);
  box-shadow: 0 0 10px rgba($accent, 0.14), 0 6px 18px rgba(7, 8, 17, 0.32);
}

@mixin glass-popup($accent) {
  background: rgba(17, 17, 27, 0.72);
  border: 1px solid rgba($accent, 0.40);
  box-shadow: 0 0 14px rgba($accent, 0.16), 0 20px 48px rgba(7, 8, 17, 0.58);
}
```

- [ ] **Step 2: Restyle the islands**

Replace the `.bar-island` and `.bar-island:hover` rules (currently `background: rgba(24, 24, 37, 0.88); border: 1px solid rgba(205, 214, 244, 0.09); …`) with:

```scss
.bar-island {
  @include glass-island($overlay1);
  border-radius: 9px;
  margin: 0 3px;
  min-height: 30px;
  padding: 0 6px;
}

.bar-island:hover {
  background: rgba(30, 30, 46, 0.66);
}

.workspace-island { @include glass-island($green); }
.media-island { @include glass-island($teal); }
.window-island { @include glass-island($lavender); }
.sys-island { @include glass-island($mauve); }
.session-island { @include glass-island($blue); }
```

Keep the existing padding-only rules for `.workspace-island`, `.media-island`, `.window-island`, `.tray-island`, `.sys-island`, `.session-island` (the `@include` lines above are *additions to* those selectors or merged into them — one rule per selector, paddings preserved). `.clock-island` currently sets its own `background: rgba(24, 24, 37, 0.88)`; replace that rule's background line with `@include glass-island($green);` keeping `padding: 0 13px;`.

- [ ] **Step 3: Restyle the popups**

Replace the shared `.popup` background/border/box-shadow (keep `border-radius: 12px; color: $text;`):

```scss
.popup {
  @include glass-popup($lavender);
  border-radius: 12px;
  color: $text;
}

.accent-volume { @include glass-popup($yellow); }
.accent-bluetooth { @include glass-popup($sapphire); }
.accent-network { @include glass-popup($peach); }
.accent-battery { @include glass-popup($blue); }
```

Apply the same treatment to the two bespoke popups (replace only their background/border/box-shadow lines, keep everything else):

```scss
.display-mode-popup {
  @include glass-popup($yellow);
  border-radius: 12px;
  color: $text;
}

.ai-usage-popup {
  @include glass-popup($sapphire);
  border-radius: 12px;
  color: $text;
}
```

Note: `.accent-*` rules must appear **after** `.popup` in the file so they win the cascade at equal specificity. `.popup-backdrop` stays `background: transparent;` — do not add blur or color to it.

- [ ] **Step 4: Blur layerrules in `modules/desktop/home/hyprland/default.nix`**

Replace the `layerrule` attribute (currently waybar + `eww-bar` only) with:

```nix
          layerrule =
            lib.optionals config.home.desktop.hyprland.waybar.enable [
              "match:namespace waybar, blur on"
            ]
            ++ lib.optionals config.home.desktop.hyprland.eww.enable (
              map (ns: "match:namespace ${ns}, blur on") [
                "eww-bar"
                "eww-volume"
                "eww-display-mode"
                "eww-bluetooth"
                "eww-network"
                "eww-battery"
                "eww-ai-usage"
                "eww-notifications"
              ]
            )
            ++ lib.optionals config.home.desktop.hyprland.dunst.enable [
              "match:namespace notifications, blur on"
            ];
```

(`eww-popup-backdrop` is deliberately excluded — blurring a fully transparent full-screen layer is undefined-looking. `eww-notifications` is defined in Task 6; a blur rule for a not-yet-existing namespace is inert. `notifications` is dunst's verified layer namespace.)

- [ ] **Step 5: Verify nix syntax + run the suite (must stay 48 passed)**

Run: `nix-instantiate --parse modules/desktop/home/hyprland/default.nix >/dev/null && echo OK`
Expected: `OK`
Run: `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py' 2>&1 | tail -1`
Expected: `OK` (48 tests)

- [ ] **Step 6: Commit**

```bash
git add modules/desktop/home/hyprland/eww/eww.scss modules/desktop/home/hyprland/default.nix
git commit -m "feat(eww): luminous-glass restyle for islands and popups"
```

---

### Task 3: Notifications backend module (pure parsing + state)

**Files:**
- Create: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/notifications.py`
- Test: `tests/eww_bar_backend/test_notifications.py`

**Interfaces:**
- Produces: `notifications_state() -> dict` shaped `{"paused": "false", "new": 0, "count": 0, "groups": [{"app", "count", "collapsed", "items": [{"id", "summary", "body", "age", "urgency"}]}]}`; pure helpers `parse_history_items(history_json)`, `format_age(seconds)`, `notifications_state_from_parts(items, paused_text, now_boot_us, collapsed, last_seen_us)`; module constant `NOTIFICATIONS_DEFAULT`. Action functions come in Task 4.

- [ ] **Step 1: Write the failing tests**

Create `tests/eww_bar_backend/test_notifications.py`:

```python
import json
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import notifications  # noqa: E402


def wrap(value):
    return {"type": "s", "data": value}


def history_fixture(items):
    return json.dumps({"type": "aa{sv}", "data": [items]})


ITEM_OLD = {
    "appname": wrap("Element"),
    "summary": wrap("Alice"),
    "body": wrap("check this out"),
    "id": {"type": "u", "data": 3},
    "timestamp": {"type": "x", "data": 100_000_000},
    "urgency": wrap("NORMAL"),
}
ITEM_NEW = {
    "appname": wrap("Element"),
    "summary": wrap("Alice"),
    "body": wrap("are we still on\nfor tonight?"),
    "id": {"type": "u", "data": 4},
    "timestamp": {"type": "x", "data": 200_000_000},
    "urgency": wrap("NORMAL"),
}
ITEM_POWER = {
    "appname": wrap("Power"),
    "summary": wrap("Battery"),
    "body": wrap("Fully charged"),
    "id": {"type": "u", "data": 5},
    "timestamp": {"type": "x", "data": 150_000_000},
    "urgency": wrap("CRITICAL"),
}


class ParseHistoryTests(unittest.TestCase):
    def test_parses_and_sorts_newest_first(self):
        items = notifications.parse_history_items(
            history_fixture([ITEM_OLD, ITEM_NEW, ITEM_POWER])
        )
        self.assertEqual([item["id"] for item in items], [4, 5, 3])
        self.assertEqual(items[0]["app"], "Element")
        self.assertEqual(items[1]["urgency"], "CRITICAL")

    def test_garbage_inputs_yield_empty(self):
        self.assertEqual(notifications.parse_history_items(""), [])
        self.assertEqual(notifications.parse_history_items("not json"), [])
        self.assertEqual(notifications.parse_history_items('{"data": {}}'), [])
        self.assertEqual(notifications.parse_history_items('{"data": [[{"id": {"data": "x"}}]]}'), [])


class FormatAgeTests(unittest.TestCase):
    def test_boundaries(self):
        self.assertEqual(notifications.format_age(0), "now")
        self.assertEqual(notifications.format_age(9.9), "now")
        self.assertEqual(notifications.format_age(59), "59s")
        self.assertEqual(notifications.format_age(60), "1m")
        self.assertEqual(notifications.format_age(3599), "59m")
        self.assertEqual(notifications.format_age(3600), "1h")
        self.assertEqual(notifications.format_age(86_399), "23h")
        self.assertEqual(notifications.format_age(200_000), "2d")


class StateFromPartsTests(unittest.TestCase):
    def build(self, collapsed=frozenset(), last_seen=0, paused="false\n"):
        items = notifications.parse_history_items(
            history_fixture([ITEM_OLD, ITEM_NEW, ITEM_POWER])
        )
        return notifications.notifications_state_from_parts(
            items, paused, 260_000_000, set(collapsed), last_seen
        )

    def test_groups_ordered_by_newest_item(self):
        state = self.build()
        self.assertEqual([group["app"] for group in state["groups"]], ["Element", "Power"])
        self.assertEqual(state["groups"][0]["count"], 2)
        self.assertEqual(state["count"], 3)

    def test_body_newlines_flattened_and_ages_formatted(self):
        state = self.build()
        top = state["groups"][0]["items"][0]
        self.assertEqual(top["body"], "are we still on for tonight?")
        self.assertEqual(top["age"], "1m")

    def test_collapsed_and_new_count(self):
        state = self.build(collapsed={"Element"}, last_seen=150_000_000)
        self.assertEqual(state["groups"][0]["collapsed"], "true")
        self.assertEqual(state["groups"][1]["collapsed"], "false")
        self.assertEqual(state["new"], 1)  # only ITEM_NEW is newer than last_seen

    def test_paused_parsing(self):
        self.assertEqual(self.build(paused="true\n")["paused"], "true")
        self.assertEqual(self.build(paused="")["paused"], "false")

    def test_truncation(self):
        long_item = {
            "appname": wrap("A" * 40),
            "summary": wrap("S" * 60),
            "body": wrap("B" * 90),
            "id": {"type": "u", "data": 9},
            "timestamp": {"type": "x", "data": 1},
            "urgency": wrap("LOW"),
        }
        items = notifications.parse_history_items(history_fixture([long_item]))
        state = notifications.notifications_state_from_parts(items, "false", 2, set(), 0)
        group = state["groups"][0]
        self.assertEqual(len(group["app"]), 20)
        self.assertEqual(len(group["items"][0]["summary"]), 48)
        self.assertEqual(len(group["items"][0]["body"]), 64)


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd tests/eww_bar_backend && python3 -m unittest test_notifications -v 2>&1 | tail -3`
Expected: FAIL/ERROR with `No module named 'eww_bar_backend.notifications'`

- [ ] **Step 3: Implement `notifications.py`**

```python
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py' 2>&1 | tail -1`
Expected: `OK` (48 + 8 new = 56 tests)
Run: `python3 -m py_compile modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/notifications.py && echo OK`
Expected: `OK`

- [ ] **Step 5: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/notifications.py tests/eww_bar_backend/test_notifications.py
git commit -m "feat(eww): notifications collector parsing dunstctl history"
```

---

### Task 4: Notif control verbs + watcher wiring

**Files:**
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/notifications.py` (action functions)
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/control.py` (verbs + payload parsing + usage line)
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/app.py` (initial state + watcher thread)
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/state.py` (seed key)
- Modify: `modules/desktop/home/hyprland/eww/default.nix` (add `dbus` to `runtimePackages`)
- Test: `tests/eww_bar_backend/test_notifications.py` (extend)

**Interfaces:**
- Consumes: Task 3's `notifications_state()` / `_UI_LOCK` / `_COLLAPSED` / `_LAST_SEEN_US`.
- Produces: `eww-barctl notif toggle-group <app> | dismiss <id> | clear-group <app> | clear-all | dnd-toggle | mark-seen`; `bar_state.notifications` live in the emitted JSON; action functions `toggle_group(app)`, `dismiss_notification(notif_id)`, `clear_group(app)`, `clear_all_notifications()`, `toggle_dnd()`, `mark_seen()` — each returns a fresh `notifications_state()`.

- [ ] **Step 1: Write the failing tests (append to `test_notifications.py`)**

```python
from eww_bar_backend import control  # noqa: E402


class NotifActionTests(unittest.TestCase):
    def test_toggle_group_flips_membership(self):
        notifications._COLLAPSED.clear()
        calls = []
        original = notifications.notifications_state
        notifications.notifications_state = lambda: calls.append(1) or {"stub": True}
        try:
            notifications.toggle_group("Element")
            self.assertIn("Element", notifications._COLLAPSED)
            notifications.toggle_group("Element")
            self.assertNotIn("Element", notifications._COLLAPSED)
        finally:
            notifications.notifications_state = original
        self.assertEqual(len(calls), 2)

    def test_toggle_group_requires_app(self):
        with self.assertRaises(ValueError):
            notifications.toggle_group("")

    def test_dismiss_requires_numeric_id(self):
        with self.assertRaises(ValueError):
            notifications.dismiss_notification("abc")


class NotifPayloadTests(unittest.TestCase):
    def test_payloads(self):
        self.assertEqual(
            control.control_payload_from_args(["notif", "toggle-group", "Claude", "Code"]),
            {"command": "notif", "action": "toggle-group", "app": "Claude Code"},
        )
        self.assertEqual(
            control.control_payload_from_args(["notif", "dismiss", "42"]),
            {"command": "notif", "action": "dismiss", "id": "42"},
        )
        self.assertEqual(
            control.control_payload_from_args(["notif", "clear-all"]),
            {"command": "notif", "action": "clear-all"},
        )
        self.assertEqual(
            control.control_payload_from_args(["notif", "dnd-toggle"]),
            {"command": "notif", "action": "dnd-toggle"},
        )

    def test_notif_without_action_is_usage_error(self):
        with self.assertRaises(ValueError):
            control.control_payload_from_args(["notif"])
```

- [ ] **Step 2: Run to verify failure**

Run: `cd tests/eww_bar_backend && python3 -m unittest test_notifications -v 2>&1 | tail -3`
Expected: FAIL (`toggle_group` missing; payload branch missing)

- [ ] **Step 3: Add action functions to `notifications.py`**

```python
def _run_dunstctl(args):
    subprocess.run(
        ["dunstctl", *args],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        check=False,
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
    newest = items[0]["timestamp"] if items else int(uptime_seconds() * 1_000_000)
    with _UI_LOCK:
        _LAST_SEEN_US = max(_LAST_SEEN_US, newest)
    return notifications_state()
```

- [ ] **Step 4: Wire `control.py`**

Import at the top (alongside the existing collector imports):

```python
from .notifications import (
    clear_all_notifications,
    clear_group,
    dismiss_notification,
    mark_seen,
    toggle_dnd,
    toggle_group,
)
```

Extend `CONTROL_USAGE` (append before the closing paren): `" | notif toggle-group <app>|dismiss <id>|clear-group <app>|clear-all|dnd-toggle|mark-seen"`.

In `handle_control_command`, before the final `raise`:

```python
    if command == "notif":
        action = payload.get("action", "")
        if action == "toggle-group":
            value = toggle_group(payload.get("app", ""))
        elif action == "dismiss":
            value = dismiss_notification(payload.get("id", ""))
        elif action == "clear-group":
            value = clear_group(payload.get("app", ""))
        elif action == "clear-all":
            value = clear_all_notifications()
        elif action == "dnd-toggle":
            value = toggle_dnd()
        elif action == "mark-seen":
            value = mark_seen()
        else:
            raise ValueError(
                "notif action must be toggle-group, dismiss, clear-group, clear-all, dnd-toggle, or mark-seen"
            )
        state.update(notifications=value)
        return {"ok": True, "command": "notif", "action": action, "notifications": value}
```

In `control_payload_from_args`, before the final `raise` (app names may contain spaces — join the tail):

```python
    if args[0] == "notif" and len(args) >= 2:
        payload = {"command": "notif", "action": args[1]}
        if args[1] in ("toggle-group", "clear-group") and len(args) >= 3:
            payload["app"] = " ".join(args[2:])
        elif args[1] == "dismiss" and len(args) >= 3:
            payload["id"] = args[2]
        return payload
```

- [ ] **Step 5: Wire `state.py` and `app.py`**

`state.py`: import `NOTIFICATIONS_DEFAULT` from `.notifications` and seed after `"tray_count": 0,`:

```python
            "notifications": dict(NOTIFICATIONS_DEFAULT, groups=[]),
```

`app.py`: add `from .notifications import notifications_state`; in `run_bar`'s initial `state.update(...)` add `notifications=notifications_state(),`; append two threads to the `threads` list (dbus watcher reuses `watch_command`'s debounce/respawn loop, and a slow-poll safety net rides the existing 30s battery thread):

```python
        threading.Thread(
            target=watch_command,
            args=(
                state,
                "notifications",
                notifications_state,
                ["dbus-monitor", "--profile", "interface='org.freedesktop.Notifications'"],
            ),
            daemon=True,
        ),
```

and change the existing 30s battery periodic to:

```python
        threading.Thread(
            target=periodic,
            args=(state, 30),
            kwargs={"battery": battery_state, "notifications": notifications_state},
            daemon=True,
        ),
```

`eww/default.nix`: add `dbus` to `runtimePackages` (alphabetical, after `coreutils`).

- [ ] **Step 6: Run the suite + compile checks**

Run: `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py' 2>&1 | tail -1`
Expected: `OK` (61 tests)
Run: `python3 -m py_compile modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/{notifications,control,app,state}.py && echo OK`
Expected: `OK`

- [ ] **Step 7: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend tests/eww_bar_backend/test_notifications.py modules/desktop/home/hyprland/eww/default.nix
git commit -m "feat(eww): notif control verbs + dbus-driven refresh"
```

---

### Task 5: `eww-popup` toggle without explicit screen

**Files:**
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/popups.py`
- Test: `tests/eww_bar_backend/test_popups.py` (extend)

**Interfaces:**
- Consumes: existing `run_popup(argv, active_windows_fn, eww_fn)` / `popup_eww_calls`.
- Produces: `eww-popup toggle <window>` (screen omitted) resolves the focused monitor via `hyprctl monitors -j` — required by the Hyprland keybinds in Task 7 and the wallpaper plan's `SUPER+W`. New injectable `monitor_fn` parameter on `run_popup`; `POPUP_USAGE` becomes `"usage: eww-popup toggle <window> [screen] | close"`.

- [ ] **Step 1: Write the failing tests (extend `test_popups.py`, matching its existing style)**

```python
class FocusedMonitorTests(unittest.TestCase):
    def test_picks_focused_monitor(self):
        text = '[{"name": "eDP-1", "focused": false}, {"name": "HDMI-A-1", "focused": true}]'
        self.assertEqual(popups.focused_monitor_from_json(text), "HDMI-A-1")

    def test_falls_back_to_zero(self):
        self.assertEqual(popups.focused_monitor_from_json(""), "0")
        self.assertEqual(popups.focused_monitor_from_json("[]"), "0")
        self.assertEqual(popups.focused_monitor_from_json('[{"name": "eDP-1"}]'), "0")


class SingleArgToggleTests(unittest.TestCase):
    def test_toggle_without_screen_uses_focused_monitor(self):
        calls = []
        rc = popups.run_popup(
            ["toggle", "volume_popup"],
            active_windows_fn=lambda: "",
            eww_fn=calls.append,
            monitor_fn=lambda: "HDMI-A-1",
        )
        self.assertEqual(rc, 0)
        self.assertIn(["open", "volume_popup", "--screen", "HDMI-A-1"], calls)
```

- [ ] **Step 2: Run to verify failure**

Run: `cd tests/eww_bar_backend && python3 -m unittest test_popups -v 2>&1 | tail -3`
Expected: FAIL (`focused_monitor_from_json` missing)

- [ ] **Step 3: Implement in `popups.py`**

Add after `parse_open_windows` (import `parse_json` alongside `run_text` from `.common`):

```python
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
```

Update `POPUP_USAGE` to `"usage: eww-popup toggle <window> [screen] | close"`. In `popup_eww_calls`, change the toggle arity check from `!= 2` to `not in (1, 2)` is NOT needed — instead resolve before the pure function. In `run_popup`, change the signature to `run_popup(argv, active_windows_fn=_active_windows_text, eww_fn=_run_eww, monitor_fn=_focused_monitor)` and insert before `popup_eww_calls`:

```python
    if command == "toggle" and len(args) == 1:
        args = [args[0], monitor_fn()]
```

(`popup_eww_calls` keeps its strict two-arg contract.)

- [ ] **Step 4: Run the suite**

Run: `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py' 2>&1 | tail -1`
Expected: `OK` (64 tests)

- [ ] **Step 5: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/popups.py tests/eww_bar_backend/test_popups.py
git commit -m "feat(eww): eww-popup toggle resolves focused monitor when screen omitted"
```

---

### Task 6: Notification center popup + bar bell

**Files:**
- Modify: `modules/desktop/home/hyprland/eww/eww.yuck` (initial JSON, bell button, new window + widgets)
- Modify: `modules/desktop/home/hyprland/eww/eww.scss` (notif classes)
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/popups.py` (`POPUP_WINDOWS`)
- Test: `tests/eww_bar_backend/test_popups.py` (extend)

**Interfaces:**
- Consumes: `bar_state.notifications` (Tasks 3–4), `glass-popup` mixin + `accent-notif` slot (Task 2), popup lifecycle (Task 5).
- Produces: window `notif_center_popup` (namespace `eww-notifications`), bell button classes `notif-bell`, `dnd`, `has-new`.

Glyphs used (paste exactly; they are nf-md bells + close):
bell `󰂚` (U+F009A), bell-off `󰂛` (U+F009B), bell-outline `󰂜` (U+F009C), close `󰅖` (U+F0156), clear-all `󰎟` (U+F039F). Group chevrons use plain text `▸`/`▾` (no nerd-font dependency).

- [ ] **Step 1: Register the window in `POPUP_WINDOWS` + failing test**

Test (append to `test_popups.py`):

```python
class NotifWindowRegisteredTests(unittest.TestCase):
    def test_notif_center_popup_is_managed(self):
        self.assertIn("notif_center_popup", popups.POPUP_WINDOWS)
```

Run: `cd tests/eww_bar_backend && python3 -m unittest test_popups -v 2>&1 | tail -3` → FAIL.
Then add `"notif_center_popup",` to the `POPUP_WINDOWS` list in `popups.py` (after `"display_mode_popup"`). Re-run → `OK` (65 tests).

- [ ] **Step 2: Extend the `deflisten` initial JSON in `eww.yuck:1`**

Insert after `"tray_count":0,`:

```
"notifications":{"paused":"false","new":0,"count":0,"groups":[]},
```

- [ ] **Step 3: Add the bell button to the bar**

In the `tray-island` box, between the systray `revealer` and the `idle-inhibitor` button, insert:

```yuck
          (button :class "module-button notif-bell ${bar_state.notifications.paused == 'true' ? 'dnd' : (bar_state.notifications.new > 0 ? 'has-new' : '')}"
            :timeout "5s"
            :onclick "eww-popup toggle notif_center_popup ${output} && eww-barctl --quiet notif mark-seen"
            :tooltip {bar_state.notifications.paused == "true" ? "Do not disturb" : "${bar_state.notifications.count} notifications"}
            (label :text {bar_state.notifications.paused == "true" ? "󰂛" : (bar_state.notifications.new > 0 ? "󰂚 ${bar_state.notifications.new}" : "󰂜")}))
```

- [ ] **Step 4: Add the popup window + widgets (append after `battery_popup`)**

```yuck
(defwindow notif_center_popup
  :monitor 0
  :geometry (geometry :x "12px" :y "50px" :width "340px" :height "420px" :anchor "top right")
  :stacking "fg"
  :exclusive false
  :focusable false
  :namespace "eww-notifications"
  (notif_center_panel))

(defwidget notif_center_panel []
  (box :class "popup accent-notif notif-center" :orientation "v" :space-evenly false
    (box :class "popup-header" :orientation "h" :space-evenly false
      (box :class "popup-mark" (label :text "󰂚"))
      (box :class "popup-title-stack" :orientation "v" :space-evenly false
        (label :class "popup-title" :halign "start" :text "Notifications")
        (label :class "popup-eyebrow" :halign "start"
          :text {bar_state.notifications.paused == "true" ? "Do not disturb" : "${bar_state.notifications.count} in history"}))
      (box :class "notif-header-actions" :orientation "h" :space-evenly false :halign "end" :hexpand true
        (button :class "notif-dnd ${bar_state.notifications.paused == 'true' ? 'active' : ''}"
          :onclick "eww-barctl --quiet notif dnd-toggle"
          :tooltip {bar_state.notifications.paused == "true" ? "Resume notifications" : "Do not disturb"}
          (label :text "󰂛"))
        (button :class "notif-clear-all"
          :onclick "eww-barctl --quiet notif clear-all"
          :tooltip "Clear all"
          (label :text "󰎟"))))
    (box :class "popup-empty notif-empty" :orientation "h" :space-evenly false
      :visible {arraylength(bar_state.notifications.groups) == 0}
      (label :halign "start" :text "No notifications"))
    (scroll :class "notif-scroll" :vscroll true :hscroll false :vexpand true
      :visible {arraylength(bar_state.notifications.groups) > 0}
      (box :class "notif-groups" :orientation "v" :space-evenly false
        (for group in {bar_state.notifications.groups}
          (notif_group :group group))))))

(defwidget notif_group [group]
  (box :class "notif-group" :orientation "v" :space-evenly false
    (box :class "notif-group-header" :orientation "h" :space-evenly false
      (button :class "notif-group-toggle" :hexpand true
        :onclick "eww-barctl --quiet notif toggle-group '${group.app}'"
        (box :orientation "h" :space-evenly false
          (label :class "notif-group-app" :halign "start" :text {group.app})
          (label :class "notif-group-count" :text {group.count})
          (label :class "notif-group-chevron" :halign "end" :hexpand true
            :text {group.collapsed == "true" ? "▸" : "▾"})))
      (button :class "notif-group-clear"
        :onclick "eww-barctl --quiet notif clear-group '${group.app}'"
        :tooltip "Clear group"
        (label :text "󰅖")))
    (revealer :transition "slidedown" :duration "200ms" :reveal {group.collapsed != "true"}
      (box :class "notif-items" :orientation "v" :space-evenly false
        (for item in {group.items}
          (notif_item :item item))))))

(defwidget notif_item [item]
  (box :class "notif-item ${item.urgency == 'CRITICAL' ? 'critical' : ''}" :orientation "h" :space-evenly false
    (box :class "notif-item-body" :orientation "v" :space-evenly false :hexpand true
      (box :orientation "h" :space-evenly false
        (label :class "notif-item-summary" :halign "start" :text {item.summary})
        (label :class "notif-item-age" :halign "end" :hexpand true :text {item.age}))
      (label :class "notif-item-text" :halign "start" :visible {item.body != ""} :text {item.body}))
    (button :class "notif-item-dismiss" :valign "start"
      :onclick "eww-barctl --quiet notif dismiss ${item.id}"
      (label :text "󰅖"))))
```

Known limitations, accepted by the spec: app names containing a single-quote would break the `'${group.app}'` shell quoting (rare; harmless failure — the click does nothing); eww's `for`-over-array re-append quirk (documented at `eww.yuck:379`) may glitch group ordering while the popup is open during a live update — evaluate at calibration (Task 8).

- [ ] **Step 5: SCSS for the center + bell (append near the other popup-specific sections)**

```scss
/* Notification center */
.accent-notif { @include glass-popup($mauve); }
.accent-notif .popup-mark { background: rgba(203, 166, 247, 0.11); border-color: rgba(203, 166, 247, 0.18); color: $mauve; }

.notif-bell.has-new { color: $mauve; }
.notif-bell.dnd { color: $overlay0; }

.notif-header-actions { margin-left: 8px; }
.notif-dnd,
.notif-clear-all {
  border-radius: 6px;
  color: $subtext0;
  margin-left: 6px;
  min-height: 28px;
  min-width: 28px;
}
.notif-dnd:hover,
.notif-clear-all:hover { background: rgba(203, 166, 247, 0.10); color: $mauve; }
.notif-dnd.active { background: rgba(203, 166, 247, 0.14); color: $mauve; }

.notif-scroll { margin: 4px 3px 6px 0; }
.notif-scroll scrollbar { background: transparent; min-width: 5px; }
.notif-scroll scrollbar slider { background: rgba(127, 132, 156, 0.34); border-radius: 999px; min-width: 3px; }

.notif-groups { padding: 6px 10px 8px 14px; }
.notif-group { margin-bottom: 8px; }
.notif-group-header { min-height: 28px; }
.notif-group-toggle { border-radius: 7px; padding: 0 8px; }
.notif-group-toggle:hover { background: rgba(49, 50, 68, 0.5); }
.notif-group-app { color: $mauve; font-size: 12px; font-weight: 800; }
.notif-group-count {
  background: rgba(203, 166, 247, 0.14);
  border-radius: 999px;
  color: $mauve;
  font-size: 9px;
  font-weight: 800;
  margin-left: 7px;
  padding: 1px 7px;
}
.notif-group-chevron { color: $overlay0; font-size: 10px; }
.notif-group-clear { border-radius: 6px; color: $overlay1; min-width: 26px; }
.notif-group-clear:hover { background: rgba(243, 139, 168, 0.14); color: $red; }

.notif-item { border-radius: 7px; margin-top: 3px; padding: 6px 8px; }
.notif-item:hover { background: rgba(49, 50, 68, 0.4); }
.notif-item-summary { color: $text; font-size: 12px; font-weight: 700; }
.notif-item-age { color: $overlay0; font-size: 9px; margin-left: 8px; }
.notif-item-text { color: $subtext0; font-size: 11px; margin-top: 1px; }
.notif-item.critical .notif-item-summary { color: $red; }
.notif-item-dismiss { border-radius: 6px; color: $overlay1; margin-left: 6px; min-width: 24px; }
.notif-item-dismiss:hover { background: rgba(243, 139, 168, 0.14); color: $red; }
.notif-empty { padding: 14px; }
```

- [ ] **Step 6: Glyph + paren sanity check**

Run: `python3 - <<'EOF'`
```python
text = open("modules/desktop/home/hyprland/eww/eww.yuck", encoding="utf-8").read()
for glyph, name in [("󰂚", "bell"), ("󰂛", "bell-off"), ("󰂜", "bell-outline"), ("󰅖", "close"), ("󰎟", "clear-all")]:
    assert glyph in text, f"missing glyph: {name}"
depth = 0
in_string = False
prev = ""
for ch in text:
    if ch == '"' and prev != "\\":
        in_string = not in_string
    elif not in_string:
        depth += ch == "("
        depth -= ch == ")"
        assert depth >= 0
    prev = ch
assert depth == 0, f"unbalanced parens: {depth}"
print("OK")
EOF
```
Expected: `OK`
Run: `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py' 2>&1 | tail -1`
Expected: `OK` (65 tests)

- [ ] **Step 7: Commit**

```bash
git add modules/desktop/home/hyprland/eww/eww.yuck modules/desktop/home/hyprland/eww/eww.scss modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/popups.py tests/eww_bar_backend/test_popups.py
git commit -m "feat(eww): notification center popup + bar bell"
```

---

### Task 7: Dunst glass restyle + config fixes + Hyprland keybinds

**Files:**
- Modify: `modules/desktop/home/hyprland/dunst.nix`
- Modify: `modules/desktop/home/hyprland/default.nix` (bind list)

**Interfaces:**
- Consumes: `palette` (Task 1), verbs (`notif dnd-toggle`, `mark-seen`) and single-arg `eww-popup toggle` (Tasks 4–5).
- Produces: final dunst config; binds `SUPER+N`, `SUPER+CTRL+N`, `SUPER+ESCAPE`, `SUPER+SHIFT+ESCAPE`.

- [ ] **Step 1: Restyle + fix `dunst.nix` global section**

Apply these exact changes inside `settings.global`:
- `offset = "16x56";` → `offset = "(16, 56)";` (legacy syntax warning)
- `height = 120;` → `height = "(0, 300)";` (legacy-height warning; dynamic up to 300px)
- Delete the `icon_size = 32;` line (invalid key, warned at startup)
- `corner_radius = 12;` stays (matches popup radius)
- `history_length = 20;` → `history_length = 50;`
- Delete `close = "ctrl+space";` and `close_all = "ctrl+shift+space";` (X11-only, dead on Wayland — replaced by Hyprland binds below)
- Add `highlight = palette.mauve;` (progress-bar fill matches the accent language)

- [ ] **Step 2: Glass urgency colors (alpha-suffixed hex; dunst ≥1.8 supports `#RRGGBBAA` on Wayland)**

```nix
        urgency_low = {
          background = "${palette.crust}99";
          foreground = palette.subtext0;
          frame_color = "${palette.overlay0}40";
          timeout = 4;
        };

        urgency_normal = {
          background = "${palette.crust}99";
          foreground = palette.text;
          frame_color = "${palette.mauve}73";
          timeout = 6;
        };

        urgency_critical = {
          background = "${palette.crust}b3";
          foreground = palette.red;
          frame_color = "${palette.red}b3";
          timeout = 0; # never auto-dismiss
        };
```

(`99` ≈ 60%, `73` ≈ 45%, `b3` ≈ 70% — the same recipe as the SCSS glass mixins; the blur layerrule for namespace `notifications` was added in Task 2.)

- [ ] **Step 3: Keybinds in `modules/desktop/home/hyprland/default.nix`**

After the existing `lib.optionals (… eww.enable && … laptopControls.enable) [ … ]` block in the `bind` list, add two more blocks:

```nix
          ++ lib.optionals config.home.desktop.hyprland.eww.enable [
            "${MOD1}, n, exec, eww-popup toggle notif_center_popup && eww-barctl --quiet notif mark-seen"
            "${MOD1}+CTRL, n, exec, eww-barctl --quiet notif dnd-toggle"
          ]
          ++ lib.optionals config.home.desktop.hyprland.dunst.enable [
            "${MOD1}, Escape, exec, dunstctl close"
            "${MOD1}+SHIFT, Escape, exec, dunstctl close-all"
          ]
```

(`eww-popup toggle` with no screen resolves the focused monitor — Task 5. `ctrl+space` untouched: fcitx owns it.)

- [ ] **Step 4: Verify syntax**

Run: `nix-instantiate --parse modules/desktop/home/hyprland/dunst.nix >/dev/null && nix-instantiate --parse modules/desktop/home/hyprland/default.nix >/dev/null && echo OK`
Expected: `OK`

- [ ] **Step 5: Commit**

```bash
git add modules/desktop/home/hyprland/dunst.nix modules/desktop/home/hyprland/default.nix
git commit -m "feat(dunst): luminous-glass toasts + Wayland keybinds"
```

---

### Task 8: Calibration + live verification (CONTROLLER ONLY)

**Files:** possibly small tweaks to `eww.scss` / `eww.yuck` geometry values.

This task is executed by the controller session after the **user** triggers `NIXHOST=laptop just build` and `systemctl --user restart eww-bar` (and dunst restarts via HM). Do not dispatch to an implementer subagent.

- [ ] Ask the user to rebuild when RAM is free; wait for confirmation.
- [ ] `dunstctl history-clear`, then `notify-send -a TestApp "Summary" "Body"` ×3 (one `-u critical`) — toasts render glass (translucent + blur + mauve/red frame), no dunst startup warnings in `journalctl --user -u dunst -b`.
- [ ] Bell shows count; `eww-popup toggle notif_center_popup` and `SUPER+N` open the center; groups collapse/expand; per-item ✕ and group-clear work (`dunstctl count history` drops); clear-all empties; DND toggle flips bell glyph + `dunstctl is-paused`; mark-seen resets the badge.
- [ ] Blur/transparency check on bar islands + all popups (grim screenshot, read it); verify backdrop still closes popups; verify the `for`-loop quirk doesn't corrupt the group list on live updates (send a notification while the center is open).
- [ ] `SUPER+ESCAPE` / `SUPER+SHIFT+ESCAPE` dismiss toasts.
- [ ] Tune calibration values only if something looks off (island/popup alphas, glow radii, center height/width, bell placement), one rebuild-free `eww reload` iteration at a time where possible; commit as `fix(eww): glass calibration pass`.
- [ ] Update `.superpowers/sdd/progress.md` with results.

---

## Verification checklist (spec → task)

- Palette extraction → Task 1. Glass bar/popups + blur rules → Task 2. Collector/pipeline/dbus → Tasks 3–4. Verbs table → Task 4. Popup + L2 layout + scroll + empty state → Task 6. Bell + badge semantics → Tasks 4, 6. Toast restyle + warnings + history 50 → Task 7. Keybinds → Task 7 (Hyprland) + Task 5 (screen-less toggle). Error handling → `run_text`/`check=False` throughout; watcher respawn via `watch_command` (Task 4). Testing strategy → per-task tests + Task 8 live pass.
