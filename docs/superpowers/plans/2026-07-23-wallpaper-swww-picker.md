# Wallpaper System (swww + Glass Grid Picker) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the active wallpaper engine with swww (animated transitions, GIF wallpapers), a runtime `~/Pictures/wallpapers` collection, and a glass grid picker popup on `SUPER+W` — keeping hyprpaper as the option-gated fallback engine.

**Architecture:** A new `wallpaper/` HM module runs `swww-daemon` + a first-boot seed oneshot; option pair `swww.enable` / `hyprpaper.enable` makes the engines mutually exclusive (waybar-vs-eww precedent). A new `wallpaper.py` backend module scans the collection, caches `magick` thumbnails by content hash, parses `swww query`, and pre-chunks the grid into rows of 3 (eww has no wrapping container). The picker is one more window in the existing popup system.

**Tech Stack:** swww 0.12.1, imagemagick (thumbnails + build-time JXL→PNG seed), eww 0.6.0, Python 3 stdlib, Home Manager systemd user services.

**Spec:** `docs/superpowers/specs/2026-07-23-wallpaper-swww-picker-design.md`

## Global Constraints

- **Depends on the notifications plan being merged first**: reuses `glass-popup` SCSS mixin, `@import "palette"`, and single-arg `eww-popup toggle` (focused-monitor default).
- Implementers must NOT run `sudo`, `just build`, `nixos-rebuild`, or `systemctl`. The user triggers rebuilds; the controller verifies live (Task 5).
- Python stdlib only; test suite `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py'` must stay green (baseline: 65 after the notifications plan).
- **Engine rename (discovered in Task 1 review):** the pinned nixpkgs renamed swww → awww; `pkgs.swww` is a deprecated alias to `awww-0.12.1` whose bin dir ships ONLY `awww` and `awww-daemon`. All package references, binary invocations, unit names, and option names use `awww`. The CLI is the renamed continuation of swww (same subcommands); output format assumptions verified live in Task 5.
- awww-supported extensions only: `.jpg .jpeg .png .gif .webp .bmp .tiff` — **never `.jxl`** (awww cannot decode it).
- Booleans inside `bar_state` are strings (`"true"`/`"false"`).
- Names that are load-bearing across tasks: window `wallpaper_picker_popup` (namespace `eww-wallpaper`), state key `wallpaper`, verb group `wallpaper`, option `home.desktop.hyprland.awww.enable`, collection dir `~/Pictures/wallpapers`, seed file `pixel_sunset.png`.
- Commit after every task.

---

### Task 1: Engine options + swww module (daemon, seed, init)

**Files:**
- Modify: `modules/desktop/home/options.nix` (two new options)
- Modify: `modules/desktop/home/hyprland/hyprpaper/default.nix:7` (gate on its own option)
- Create: `modules/desktop/home/hyprland/wallpaper/default.nix`
- Modify: `modules/desktop/home/hyprland/default.nix:8-20` (imports list)
- Modify: `profiles/laptop/config.nix`, `profiles/desktop/config.nix` (enable swww)

**Interfaces:**
- Produces: options `home.desktop.hyprland.awww.enable` (default `false`) and `home.desktop.hyprland.hyprpaper.enable` (default `hyprland.enable && !awww.enable`), each also mirrored as a NixOS-level option in `modules/desktop/options.nix` (profiles assign at system level); systemd user service `awww-daemon` (+ seed init); `~/Pictures/wallpapers/pixel_sunset.png` present on activation. Later tasks assume `awww` on the backend PATH (Task 3 adds it to eww `runtimePackages`).

- [ ] **Step 1: Add the options (in `options.nix`, after the `hyprland.dunst.enable` block, same `mkHomeOpt` pattern)**

```nix
      hyprland.awww.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.awww.enable";
        default = false;
        description = "Enable the awww (formerly swww) wallpaper engine and picker for Hyprland";
      };

      hyprland.hyprpaper.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.hyprpaper.enable";
        default = config.home.desktop.hyprland.enable && !config.home.desktop.hyprland.awww.enable;
        description = "Enable hyprpaper static wallpaper (fallback engine when awww is off)";
      };
```

- [ ] **Step 2: Re-gate hyprpaper (contents untouched)**

In `hyprpaper/default.nix` change only the `mkIf` condition:

```nix
  config = lib.mkIf config.home.desktop.hyprland.hyprpaper.enable {
```

- [ ] **Step 3: Create `wallpaper/default.nix`**

```nix
{
  lib,
  config,
  pkgs,
  ...
}:
let
  cfg = config.home.desktop.hyprland;

  # Repo source of truth stays JXL (hyprpaper still uses it); awww cannot
  # decode JXL, so derive a PNG seed at build time.
  seedPng = pkgs.runCommand "pixel-sunset-png" { nativeBuildInputs = [ pkgs.imagemagick ]; } ''
    magick ${../hyprpaper/wallpaper/pixel_sunset.jxl} png:$out
  '';

  awwwInit = pkgs.writeShellScript "awww-init" ''
    export PATH=${lib.makeBinPath [ pkgs.awww pkgs.gnugrep pkgs.coreutils ]}
    for _ in $(seq 1 50); do
      if awww query >/dev/null 2>&1; then
        break
      fi
      sleep 0.1
    done
    # awww restores its own per-output cache on start; only seed a blank slate.
    if ! awww query 2>/dev/null | grep -q "image: "; then
      awww img "$HOME/Pictures/wallpapers/pixel_sunset.png" --transition-type none
    fi
  '';
in
{
  config = lib.mkIf cfg.awww.enable {
    home.packages = [ pkgs.awww ];

    home.file."Pictures/wallpapers/pixel_sunset.png".source = seedPng;

    systemd.user.services.awww-daemon = {
      Unit = {
        Description = "awww wallpaper daemon";
        After = [ "hyprland-session.target" ];
        PartOf = [ "hyprland-session.target" ];
      };

      Service = {
        ExecStart = "${pkgs.awww}/bin/awww-daemon";
        ExecStartPost = awwwInit;
        Restart = "on-failure";
        RestartSec = "1s";
      };

      Install.WantedBy = [ "hyprland-session.target" ];
    };
  };
}
```

- [ ] **Step 4: Import the module + enable in profiles**

`modules/desktop/home/hyprland/default.nix` imports list: add `./wallpaper` after `./hyprpaper`.

`profiles/laptop/config.nix` and `profiles/desktop/config.nix`: next to the existing `home.desktop.hyprland.eww.enable = true;` line add:

```nix
  home.desktop.hyprland.awww.enable = true;
```

- [ ] **Step 5: Verify option wiring evaluates**

Run: `nix-instantiate --parse modules/desktop/home/hyprland/wallpaper/default.nix >/dev/null && nix-instantiate --parse modules/desktop/home/options.nix >/dev/null && echo OK`
Expected: `OK`
Run: `nix flake check --no-build 2>&1 | tail -2` (if it errors on unrelated pre-existing issues, at minimum `nix eval .#nixosConfigurations --apply builtins.attrNames` must still evaluate)
Expected: no new evaluation errors mentioning `swww`, `hyprpaper`, or `wallpaper`

- [ ] **Step 6: Commit**

```bash
git add modules/desktop/home/options.nix modules/desktop/home/hyprland/hyprpaper/default.nix modules/desktop/home/hyprland/wallpaper modules/desktop/home/hyprland/default.nix profiles/laptop/config.nix profiles/desktop/config.nix
git commit -m "feat(wallpaper): swww engine module with hyprpaper fallback options"
```

---

### Task 2: Wallpaper backend module

**Files:**
- Create: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/wallpaper.py`
- Test: `tests/eww_bar_backend/test_wallpaper.py`

**Interfaces:**
- Produces: `wallpaper_state() -> {"current": str, "count": int, "rows": [[{"name","path","thumb","animated","active"}]]}` (rows pre-chunked to 3 columns); `set_wallpaper(path)` (applies via awww + returns fresh state); pure helpers `scan_wallpaper_files(directory)`, `parse_awww_query(text)`, `thumb_cache_path(path, stat)`, `wallpaper_items(files, current, thumb_fn)`, `rows_from_items(items)`; constant `WALLPAPER_DEFAULT`.

- [ ] **Step 1: Write the failing tests**

Create `tests/eww_bar_backend/test_wallpaper.py`:

```python
import sys
import tempfile
import unittest
from collections import namedtuple
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import wallpaper  # noqa: E402

FakeStat = namedtuple("FakeStat", "st_mtime_ns st_size")


class ScanTests(unittest.TestCase):
    def test_filters_supported_extensions_sorted(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name in ("b.png", "a.jpg", "anim.gif", "skip.jxl", "note.txt", "pic.webp"):
                (root / name).write_bytes(b"x")
            (root / "subdir").mkdir()
            files = wallpaper.scan_wallpaper_files(root)
            self.assertEqual(
                [f.name for f in files], ["a.jpg", "anim.gif", "b.png", "pic.webp"]
            )

    def test_missing_directory_yields_empty(self):
        self.assertEqual(wallpaper.scan_wallpaper_files("/nonexistent/nowhere"), [])


class QueryParseTests(unittest.TestCase):
    def test_parses_image_path(self):
        text = 'eDP-1: 1920x1200, scale: 2, currently displaying: image: /home/u/Pictures/wallpapers/pixel_sunset.png\n'
        self.assertEqual(
            wallpaper.parse_awww_query(text),
            "/home/u/Pictures/wallpapers/pixel_sunset.png",
        )

    def test_color_or_garbage_yields_empty(self):
        self.assertEqual(wallpaper.parse_awww_query("eDP-1: ... displaying: color: 000000"), "")
        self.assertEqual(wallpaper.parse_awww_query(""), "")


class ThumbCacheTests(unittest.TestCase):
    def test_stable_hash_from_path_mtime_size(self):
        path = Path("/x/y.png")
        first = wallpaper.thumb_cache_path(path, FakeStat(1, 2))
        second = wallpaper.thumb_cache_path(path, FakeStat(1, 2))
        changed = wallpaper.thumb_cache_path(path, FakeStat(9, 2))
        self.assertEqual(first, second)
        self.assertNotEqual(first, changed)
        self.assertTrue(str(first).endswith(".png"))


class ItemsAndRowsTests(unittest.TestCase):
    def test_items_flags(self):
        files = [Path("/w/sunset.png"), Path("/w/loop.gif")]
        items = wallpaper.wallpaper_items(files, "/w/loop.gif", thumb_fn=lambda p: f"/thumbs/{p.name}")
        self.assertEqual(items[0]["active"], "false")
        self.assertEqual(items[0]["animated"], "false")
        self.assertEqual(items[1]["active"], "true")
        self.assertEqual(items[1]["animated"], "true")
        self.assertEqual(items[0]["name"], "sunset")
        self.assertEqual(items[0]["thumb"], "/thumbs/sunset.png")

    def test_rows_chunking(self):
        items = list(range(7))
        rows = wallpaper.rows_from_items(items)
        self.assertEqual(rows, [[0, 1, 2], [3, 4, 5], [6]])
        self.assertEqual(wallpaper.rows_from_items([]), [])


class SetWallpaperTests(unittest.TestCase):
    def test_empty_path_raises(self):
        with self.assertRaises(ValueError):
            wallpaper.set_wallpaper("")


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run to verify failure**

Run: `cd tests/eww_bar_backend && python3 -m unittest test_wallpaper -v 2>&1 | tail -3`
Expected: FAIL with `No module named 'eww_bar_backend.wallpaper'`

- [ ] **Step 3: Implement `wallpaper.py`**

```python
import hashlib
import subprocess
from pathlib import Path

from .common import run_text, truncate_text

WALLPAPER_DEFAULT = {"current": "", "count": 0, "rows": []}

WALLPAPER_DIR = Path("~/Pictures/wallpapers").expanduser()
THUMB_DIR = Path("~/.cache/eww-bar/wallpaper-thumbs").expanduser()
# awww-decodable only — notably NOT .jxl.
EXTENSIONS = {".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff"}
GRID_COLUMNS = 3
NAME_MAX = 22
# Calibration values; grow-from-top-right matches where the picker popup sits.
AWWW_TRANSITION = [
    "--transition-type", "grow",
    "--transition-pos", "top-right",
    "--transition-duration", "0.8",
    "--transition-fps", "60",
]


def scan_wallpaper_files(directory=None):
    directory = Path(directory) if directory else WALLPAPER_DIR
    try:
        entries = sorted(directory.iterdir())
    except Exception:
        return []
    return [
        entry for entry in entries
        if entry.is_file() and entry.suffix.lower() in EXTENSIONS
    ]


def parse_awww_query(text):
    # `awww query` prints one line per output, e.g.
    # `eDP-1: 1920x1200, scale: 2, currently displaying: image: /path/img.png`.
    # Both monitors mirror, so the first image path wins.
    for line in text.splitlines():
        if "image: " in line:
            return line.split("image: ", 1)[1].strip()
    return ""


def thumb_cache_path(path, stat=None):
    stat = stat or path.stat()
    digest = hashlib.sha1(
        f"{path}:{stat.st_mtime_ns}:{stat.st_size}".encode()
    ).hexdigest()
    return THUMB_DIR / f"{digest}.png"


def ensure_thumbnail(path):
    try:
        thumb = thumb_cache_path(path)
    except Exception:
        return ""
    if thumb.exists():
        return str(thumb)
    try:
        THUMB_DIR.mkdir(parents=True, exist_ok=True)
        subprocess.run(
            [
                "magick", f"{path}[0]",
                "-thumbnail", "320x200^", "-gravity", "center", "-extent", "320x200",
                str(thumb),
            ],
            stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
            check=False, timeout=15,
        )
    except Exception:
        return ""
    return str(thumb) if thumb.exists() else ""


def wallpaper_items(files, current, thumb_fn=ensure_thumbnail):
    return [
        {
            "name": truncate_text(path.stem, NAME_MAX),
            "path": str(path),
            "thumb": thumb_fn(path),
            "animated": "true" if path.suffix.lower() == ".gif" else "false",
            "active": "true" if str(path) == current else "false",
        }
        for path in files
    ]


def rows_from_items(items, columns=GRID_COLUMNS):
    return [items[i:i + columns] for i in range(0, len(items), columns)]


def wallpaper_state():
    current = parse_awww_query(run_text(["awww", "query"]))
    items = wallpaper_items(scan_wallpaper_files(), current)
    return {"current": current, "count": len(items), "rows": rows_from_items(items)}


def set_wallpaper(path):
    if not path:
        raise ValueError("wallpaper set requires a path")
    subprocess.run(
        ["awww", "img", path, *AWWW_TRANSITION],
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        check=False,
    )
    return wallpaper_state()
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py' 2>&1 | tail -1`
Expected: `OK` (65 + 8 = 73 tests)
Run: `python3 -m py_compile modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/wallpaper.py && echo OK`
Expected: `OK`

- [ ] **Step 5: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/wallpaper.py tests/eww_bar_backend/test_wallpaper.py
git commit -m "feat(eww): wallpaper backend (scan, thumbs, swww query/set)"
```

---

### Task 3: Control verbs + app wiring

**Files:**
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/control.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/app.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/state.py`
- Modify: `modules/desktop/home/hyprland/eww/default.nix` (`runtimePackages`)
- Test: `tests/eww_bar_backend/test_wallpaper.py` (extend)

**Interfaces:**
- Consumes: Task 2's `wallpaper_state()` / `set_wallpaper(path)` / `WALLPAPER_DEFAULT`.
- Produces: `eww-barctl wallpaper set <path> | rescan`; `bar_state.wallpaper` in the emitted JSON.

- [ ] **Step 1: Failing tests (append to `test_wallpaper.py`)**

```python
from eww_bar_backend import control  # noqa: E402


class WallpaperPayloadTests(unittest.TestCase):
    def test_payloads(self):
        self.assertEqual(
            control.control_payload_from_args(["wallpaper", "set", "/w/a.png"]),
            {"command": "wallpaper", "action": "set", "path": "/w/a.png"},
        )
        self.assertEqual(
            control.control_payload_from_args(["wallpaper", "rescan"]),
            {"command": "wallpaper", "action": "rescan"},
        )

    def test_wallpaper_without_action_is_usage_error(self):
        with self.assertRaises(ValueError):
            control.control_payload_from_args(["wallpaper"])
```

Run: `cd tests/eww_bar_backend && python3 -m unittest test_wallpaper -v 2>&1 | tail -3` → FAIL.

- [ ] **Step 2: Wire `control.py`**

Import: `from .wallpaper import set_wallpaper, wallpaper_state`. Extend `CONTROL_USAGE` with `" | wallpaper set <path>|rescan"`.

`handle_control_command`, before the final `raise`:

```python
    if command == "wallpaper":
        action = payload.get("action", "")
        if action == "set":
            value = set_wallpaper(payload.get("path", ""))
        elif action == "rescan":
            value = wallpaper_state()
        else:
            raise ValueError("wallpaper action must be set or rescan")
        state.update(wallpaper=value)
        return {"ok": True, "command": "wallpaper", "action": action, "wallpaper": value}
```

`control_payload_from_args`, before the final `raise`:

```python
    if args[0] == "wallpaper" and len(args) >= 2:
        payload = {"command": "wallpaper", "action": args[1]}
        if args[1] == "set" and len(args) >= 3:
            payload["path"] = args[2]
        return payload
```

- [ ] **Step 3: Wire `state.py` and `app.py`**

`state.py`: `from .wallpaper import WALLPAPER_DEFAULT` and seed after the `"notifications"` entry:

```python
            "wallpaper": dict(WALLPAPER_DEFAULT, rows=[]),
```

`app.py`: `from .wallpaper import wallpaper_state`; do **not** add it to the synchronous initial `state.update(...)` (a cold thumbnail cache would block startup); instead append to the `threads` list:

```python
        threading.Thread(
            target=lambda: state.update(wallpaper=wallpaper_state()),
            daemon=True,
        ),
```

(No watcher: the collection changes at human speed; `rescan` fires on every picker open and after every `set`.)

- [ ] **Step 4: Runtime deps in `eww/default.nix`**

Add `awww` and `imagemagick` to `runtimePackages` (alphabetical positions).

- [ ] **Step 5: Run suite + compile**

Run: `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py' 2>&1 | tail -1`
Expected: `OK` (75 tests)
Run: `python3 -m py_compile modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/{control,app,state}.py && echo OK`
Expected: `OK`

- [ ] **Step 6: Commit**

```bash
git add modules/desktop/home/hyprland/eww/scripts/eww_bar_backend tests/eww_bar_backend/test_wallpaper.py modules/desktop/home/hyprland/eww/default.nix
git commit -m "feat(eww): wallpaper control verbs + state wiring"
```

---

### Task 4: Picker popup + keybind

**Files:**
- Modify: `modules/desktop/home/hyprland/eww/eww.yuck` (initial JSON + window + widgets)
- Modify: `modules/desktop/home/hyprland/eww/eww.scss` (picker classes)
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/popups.py` (`POPUP_WINDOWS`)
- Modify: `modules/desktop/home/hyprland/default.nix` (`SUPER+W` bind + blur namespace)
- Test: `tests/eww_bar_backend/test_popups.py` (extend)

**Interfaces:**
- Consumes: `bar_state.wallpaper` (Task 3), `glass-popup` mixin, popup lifecycle + single-arg toggle.
- Produces: window `wallpaper_picker_popup` (namespace `eww-wallpaper`), classes `accent-wallpaper`, `wallpaper-cell`.

- [ ] **Step 1: Register window + failing test**

Append to `test_popups.py`:

```python
class WallpaperWindowRegisteredTests(unittest.TestCase):
    def test_wallpaper_picker_popup_is_managed(self):
        self.assertIn("wallpaper_picker_popup", popups.POPUP_WINDOWS)
```

Run → FAIL; add `"wallpaper_picker_popup",` to `POPUP_WINDOWS`; re-run → `OK` (76 tests).

- [ ] **Step 2: Initial JSON in `eww.yuck:1`**

Insert after the `"notifications":{...},` entry added by the notifications plan:

```
"wallpaper":{"current":"","count":0,"rows":[]},
```

- [ ] **Step 3: Window + widgets (append after `notif_item`)**

Glyphs (paste exactly): image `󰸉` (U+F0E09), folder `` (U+F07B), play badge `▶`.

```yuck
(defwindow wallpaper_picker_popup
  :monitor 0
  :geometry (geometry :x "12px" :y "50px" :width "340px" :height "300px" :anchor "top right")
  :stacking "fg"
  :exclusive false
  :focusable false
  :namespace "eww-wallpaper"
  (wallpaper_panel))

(defwidget wallpaper_panel []
  (box :class "popup accent-wallpaper wallpaper-picker" :orientation "v" :space-evenly false
    (box :class "popup-header" :orientation "h" :space-evenly false
      (box :class "popup-mark" (label :text "󰸉"))
      (box :class "popup-title-stack" :orientation "v" :space-evenly false
        (label :class "popup-title" :halign "start" :text "Wallpapers")
        (label :class "popup-eyebrow" :halign "start" :text "${bar_state.wallpaper.count} in ~/Pictures/wallpapers"))
      (button :class "wallpaper-open-folder" :halign "end" :hexpand true
        :onclick "hyprctl dispatch exec 'nemo ~/Pictures/wallpapers'"
        :tooltip "Open folder"
        (label :text "")))
    (box :class "popup-empty wallpaper-empty" :orientation "h" :space-evenly false
      :visible {arraylength(bar_state.wallpaper.rows) == 0}
      (label :halign "start" :text "Drop images into ~/Pictures/wallpapers"))
    (scroll :class "wallpaper-scroll" :vscroll true :hscroll false :vexpand true
      :visible {arraylength(bar_state.wallpaper.rows) > 0}
      (box :class "wallpaper-grid" :orientation "v" :space-evenly false
        (for row in {bar_state.wallpaper.rows}
          (box :class "wallpaper-row" :orientation "h" :space-evenly false
            (for item in {row}
              (wallpaper_cell :item item))))))))

(defwidget wallpaper_cell [item]
  (button :class "wallpaper-cell ${item.active == 'true' ? 'active' : ''}"
    :timeout "5s"
    :onclick "eww-barctl --quiet wallpaper set '${item.path}' && eww-popup close"
    :tooltip {item.name}
    (overlay
      (box :class "wallpaper-thumb-missing" :visible {item.thumb == ""}
        (label :text "󰸉"))
      (image :visible {item.thumb != ""} :path {item.thumb} :image-width 92 :image-height 58)
      (label :class "wallpaper-badge" :visible {item.animated == "true"}
        :halign "end" :valign "start" :text "▶"))))
```

- [ ] **Step 4: SCSS (append after the notif section)**

```scss
/* Wallpaper picker */
.accent-wallpaper { @include glass-popup($mauve); }
.accent-wallpaper .popup-mark { background: rgba(203, 166, 247, 0.11); border-color: rgba(203, 166, 247, 0.18); color: $mauve; }

.wallpaper-open-folder {
  border-radius: 6px;
  color: $subtext0;
  margin-left: 8px;
  min-height: 28px;
  min-width: 28px;
}
.wallpaper-open-folder:hover { background: rgba(203, 166, 247, 0.10); color: $mauve; }

.wallpaper-scroll { margin: 4px 3px 6px 0; }
.wallpaper-scroll scrollbar { background: transparent; min-width: 5px; }
.wallpaper-scroll scrollbar slider { background: rgba(127, 132, 156, 0.34); border-radius: 999px; min-width: 3px; }

.wallpaper-grid { padding: 8px 10px 10px 14px; }
.wallpaper-row { margin-bottom: 8px; }
.wallpaper-cell {
  border: 1px solid rgba(205, 214, 244, 0.10);
  border-radius: 8px;
  margin-right: 8px;
  padding: 2px;
}
.wallpaper-cell:hover { border-color: rgba(203, 166, 247, 0.45); }
.wallpaper-cell.active {
  border: 2px solid rgba(203, 166, 247, 0.75);
  box-shadow: 0 0 10px rgba(203, 166, 247, 0.30);
  padding: 1px;
}
.wallpaper-thumb-missing { color: $overlay0; font-size: 22px; min-height: 58px; min-width: 92px; }
.wallpaper-badge {
  color: $text;
  font-size: 9px;
  margin: 3px 5px 0 0;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.9);
}
.wallpaper-empty { padding: 14px; }
```

Note: `text-shadow` was reset globally at the top of the file; setting it here re-enables it for the badge only.

- [ ] **Step 5: Keybind + blur namespace in `modules/desktop/home/hyprland/default.nix`**

In the eww-guarded bind block (added by the notifications plan), append:

```nix
            "${MOD1}, w, exec, eww-barctl --quiet wallpaper rescan && eww-popup toggle wallpaper_picker_popup"
```

In the eww blur-namespace list (Task 2 of the notifications plan), append `"eww-wallpaper"`.

- [ ] **Step 6: Glyph + paren sanity check and suite**

Run: `python3 - <<'EOF'`
```python
text = open("modules/desktop/home/hyprland/eww/eww.yuck", encoding="utf-8").read()
for glyph, name in [("󰸉", "image"), ("", "folder"), ("▶", "play-badge")]:
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
Expected: `OK` (76 tests)
Run: `nix-instantiate --parse modules/desktop/home/hyprland/default.nix >/dev/null && echo OK`
Expected: `OK`

- [ ] **Step 7: Commit**

```bash
git add modules/desktop/home/hyprland/eww/eww.yuck modules/desktop/home/hyprland/eww/eww.scss modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/popups.py tests/eww_bar_backend/test_popups.py modules/desktop/home/hyprland/default.nix
git commit -m "feat(eww): wallpaper picker popup on SUPER+W"
```

---

### Task 5: Calibration + live verification (CONTROLLER ONLY)

Executed by the controller after the **user** triggers the rebuild. Do not dispatch to an implementer.

- [ ] Ask the user to rebuild (`NIXHOST=laptop just build`) when RAM is free; then `systemctl --user restart eww-bar` and confirm `awww-daemon` is active while hyprpaper's unit is gone (`systemctl --user status awww-daemon hyprpaper`).
- [ ] `hyprctl layers` shows the awww daemon on the background layer of both monitors; wallpaper visible (seed or restored cache).
- [ ] Drop 2–3 test images (+1 GIF) into `~/Pictures/wallpapers`; `SUPER+W` opens the picker with thumbnails; current is highlighted; GIF shows ▶.
- [ ] Click a thumbnail → grow transition from top-right, popup closes, bar stays responsive; picked GIF animates.
- [ ] Reboot-persistence: `systemctl --user restart awww-daemon` restores the same wallpaper (cache path, no seed reapply).
- [ ] Empty-dir hint: temporarily `mv` the collection aside, rescan, verify hint; restore.
- [ ] Tune calibration values if needed (cell size 92×58, popup 340×300, transition type/duration); commit as `fix(eww): wallpaper picker calibration`.
- [ ] Update `.superpowers/sdd/progress.md`.

---

## Verification checklist (spec → task)

Engine options + hyprpaper kept → Task 1. Daemon + init + seed conversion + collection dir → Task 1. Backend state/thumbs/query/rows → Task 2. Verbs + rescan triggers → Tasks 3–4. Picker popup (grid, highlight, ▶ badge, empty hint, open-folder) → Task 4. `SUPER+W`, no bar module → Task 4. Transition constants → Task 2 (`AWWW_TRANSITION`). Error handling (missing dir, query/thumb failure) → Task 2 guards. GIF-tier animation → extension flag (Task 2) + live check (Task 5).
