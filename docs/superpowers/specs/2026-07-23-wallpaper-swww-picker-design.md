# Wallpaper System Design: swww + Glass Grid Picker

## Goal

Replace the single baked-in hyprpaper wallpaper with a proper wallpaper
system:

- **swww** as the wallpaper engine — animated transitions on change, and
  animated GIF wallpapers play natively.
- A **runtime collection directory** (`~/Pictures/wallpapers`): drop an image
  in and it's available immediately, no rebuild.
- A **glass grid picker** — an eww popup in the luminous-glass language
  (thumbnail grid, current highlighted, click to apply), opened by `SUPER+W`
  only. Nothing is added to the bar.

The existing **hyprpaper module is kept**, not deleted: swww and hyprpaper
become mutually exclusive engines selected by options (the same pattern as
waybar vs eww), with hyprpaper as the fallback when swww is disabled.

Explicitly not wanted (asked and declined): timed rotation, per-monitor
wallpapers, video wallpapers (mpvpaper), wallpaper-adaptive colors (palette
stays static Catppuccin; see the notifications/refresh spec for the
`palette.nix` extraction that would enable it later).

## Current State

- `modules/desktop/home/hyprland/hyprpaper/` runs `services.hyprpaper` with
  `ipc = on` (unused) showing `pixel_sunset.jxl`, deployed to
  `~/.config/hypr/wallpaper/`. Both monitors mirror it. It is gated only by
  `home.desktop.hyprland.enable` (no option of its own yet).
- Changing wallpaper today = edit Nix + rebuild.

Environment facts verified:

- **swww 0.12.1** is in the pinned nixpkgs. There is no Home Manager
  `services.swww` module — we run the daemon with our own user service.
- swww's supported formats: jpeg, png, **gif (animated)**, pnm, tga, tiff,
  webp, bmp, farbfeld, svg. **No JXL** — the seed must be converted.
- nixpkgs `imagemagick` (7.1.2) is built with the JXL delegate (libjxl
  0.11.2), so a build-time `runCommand` conversion of the seed works.
- `SUPER+W` is unbound in `hyprland/default.nix`.
- swww caches the applied wallpaper per output and the daemon restores it on
  startup, so the selection persists across reboots without extra state.

## Engine Selection (hyprpaper kept)

Following the waybar-vs-eww precedent in `options.nix`:

- New option `home.desktop.hyprland.swww.enable` (default `false`; turned on
  in the laptop and desktop profiles alongside eww).
- New option `home.desktop.hyprland.hyprpaper.enable`, default
  `hyprland.enable && !swww.enable` — the hyprpaper module's `mkIf` moves to
  this option but its contents stay untouched. Disable swww and hyprpaper
  takes over again, exactly as before.

## Daemon & Persistence

- New module `modules/desktop/home/hyprland/wallpaper/` alongside
  `hyprpaper/`, gated by the new swww option.
- `swww-daemon` runs as a systemd user service bound to
  `graphical-session.target` (same lifecycle pattern as `eww-bar`).
- A `swww-init` oneshot runs after the daemon: if `swww query` reports no
  image on the outputs (first boot, cleared cache), apply the seed with no
  transition. Otherwise do nothing — swww's own cache restore wins.

## Collection & Seed

- Collection dir: `~/Pictures/wallpapers`, created by Home Manager.
- Seed: `pixel_sunset.jxl` stays in the repo as the source of truth; a
  `runCommand` derivation converts it to PNG with imagemagick at build time,
  and Home Manager links `pixel_sunset.png` into the collection dir.
- The picker scans for swww-decodable extensions only: `jpg jpeg png gif webp
  bmp tiff`. `.gif` entries are flagged `animated` and play natively when
  applied (CPU cost is the user's choice by picking a GIF).
- The hyprpaper module and its deployed files stay in the repo unchanged
  (fallback engine, see Engine Selection); only its enable gating moves to
  the new option.

## Backend (eww_bar_backend)

New `wallpaper` collector + control verbs, following the existing patterns:

```
{
  "current": "/home/user/Pictures/wallpapers/pixel_sunset.png",
  "count": 12,
  "items": [
    {"name": "pixel_sunset", "path": "...", "thumb": "~/.cache/...",
     "animated": false},
    ...
  ]
}
```

- `current` is parsed from `swww query` output (first output's image path;
  both monitors mirror). Query failure → `current: ""` (no highlight).
- Thumbnails: `magick '<file>[0]' -thumbnail 320x200^ ...` into
  `~/.cache/eww-bar/wallpaper-thumbs/<content-hash>.png` — `[0]` takes a
  GIF's first frame; the hash means thumbnails regenerate only for new or
  changed files. Generation happens on rescan, guarded by the timeout
  helper; a failed thumbnail falls back to a generic placeholder.
- Control verbs:

| verb | effect |
| --- | --- |
| `wallpaper set <path>` | `swww img <path>` with the standard transition |
| `wallpaper rescan` | rescan dir + refresh thumbnails + re-emit state |

- Rescan triggers: picker open (the keybind/trigger fires
  `eww-barctl --quiet wallpaper rescan` alongside the popup toggle) and
  after every `set`. No inotify watcher — the collection changes rarely and
  always at human speed.

## Picker Popup

- `defwindow wallpaper_picker_popup`, `:namespace "eww-wallpaper"`,
  registered in `POPUP_WINDOWS` → inherits the popup lifecycle: backdrop,
  click-outside close, mutual exclusion, consistent top-right corner
  (`x=12px, y=50px`).
- Luminous-glass card (mauve accent): `popup_header` with an image-glyph
  mark, "Wallpapers" title, item count, and an open-folder button
  (`nemo ~/Pictures/wallpapers`).
- Body: 3-column thumbnail grid (eww `image` widgets inside buttons),
  scrollable past ~4 rows. The current wallpaper gets a mauve accent border +
  glow; `animated` items get a small ▶ badge. Click →
  `eww-barctl --quiet wallpaper set <path>`, then the popup closes.
- Empty collection → centered hint: "Drop images into ~/Pictures/wallpapers".

## Transitions

- Standard apply: `swww img <path> --transition-type grow --transition-pos
  top-right --transition-duration 0.8 --transition-fps 60` — the new
  wallpaper grows from the picker's corner. Constants live in one place in
  the backend; `fade` is the documented fallback taste. Exact values are
  calibration-pass tunables.

## Keybind

- `SUPER+W` → toggle the picker (popup toggle + rescan), guarded by the eww
  enable option like the other eww binds. No bar module.

## Error Handling

- Missing/empty collection dir → empty state with the hint (never an error).
- `swww query`/`swww img` failures → logged, state keeps last known values;
  the popup still lists files.
- Thumbnail failures → placeholder image; scan continues.
- All external calls use the existing timeout-guarded subprocess helpers.

## Testing

- Unit tests (fixtures, no subprocesses): extension filtering + `animated`
  flagging, `swww query` parsing, thumb-cache hash naming, state shape,
  verb routing.
- `python3 -m py_compile` + eww config compile check.
- Live (after user-triggered rebuild): daemon starts and restores cache;
  `SUPER+W` opens the picker with thumbnails; click applies with the grow
  transition; GIF wallpaper animates; reboot persistence; only one engine
  running (`hyprctl layers` shows swww-daemon, not hyprpaper, on the
  background layer while `swww.enable` is set — and the reverse when it
  isn't).

## Out of Scope

- Timed rotation / shuffle.
- Per-monitor wallpapers (swww supports `--outputs`; the backend verb takes a
  path only, so this can be added later without redesign).
- Video wallpapers (mpvpaper) and shader wallpapers.
- Wallpaper-adaptive color generation (future project; enabled by
  `palette.nix` from the refresh spec).
