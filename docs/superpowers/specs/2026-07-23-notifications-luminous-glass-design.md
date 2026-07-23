# Notification System + Luminous-Glass Refresh Design

## Goal

Two coupled deliverables, one design language:

1. **Refresh the core desktop surfaces** — bar islands, every eww popup, dunst
   toasts, and the new notification center — in a "luminous glass" style:
   translucent blurred surfaces with per-module accent-tinted borders and a
   faint glow.
2. **Make notifications first-class**: keep dunst as the daemon, add an eww
   notification center (history grouped by app), a bell module on the bar with
   a new-notification badge and do-not-disturb, and working Wayland keybinds
   to replace dunst's dead X11 shortcuts.

Rofi, wlogout, and hyprlock keep their current styling (later polish pass).

## Current State

- **dunst 1.13.2** (Wayland-native) renders toasts, themed by hex values
  hardcoded in `modules/desktop/home/hyprland/dunst.nix`. Its
  `close = "ctrl+space"` / `close_all` settings are X11-only and silently dead
  under Wayland. History is kept (`history_length = 20`, sticky) but nothing
  can display it. There is no DND control anywhere.
- **eww bar** (`eww.yuck` + `eww.scss`) has 8 windows, all with explicit
  `:namespace` (`eww-bar`, `eww-popup-backdrop`, `eww-volume`,
  `eww-display-mode`, `eww-bluetooth`, `eww-network`, `eww-battery`,
  `eww-ai-usage`). Popups share a card language: opaque
  `rgba(17,17,27,0.985)` background, 12px radius, hairline white border, soft
  shadow, `popup_header` widget. All open at the consistent top-right corner
  (`x=12px, y=50px`) via the `eww-popup` helper (backdrop + toggle + tracked
  close).
- **Hyprland blur** is only enabled for the bar itself:
  `layerrule = match:namespace eww-bar, blur on` (plus waybar). Popups and
  dunst are unblurred, so they are opaque today.
- **Palette duplication**: `eww.scss` defines the Catppuccin Mocha palette as
  SCSS variables at the top; `dunst.nix` (and `hyprlock.nix`) repeat the same
  hex values independently.
- **Backend**: `scripts/eww_bar_backend/` — event-driven collector threads
  feed one `bar_state` JSON; a Unix-socket control server handles
  `eww-barctl <verb>`; `eww-popup` manages popup windows. 48 unit tests in
  `tests/eww_bar_backend/`.

Environment facts verified on the host:

- `dunstctl history` emits JSON with per-item `appname`, `summary`, `body`,
  `icon_path`, `id`, `timestamp`, `urgency`, `default_action_name`, `urls`.
  `timestamp` is **microseconds on the monotonic boot clock**, so ages must be
  computed against `/proc/uptime`, not wall time.
- `dunstctl` verbs available: `set-paused toggle|true|false`, `is-paused`,
  `close`, `close-all`, `history-pop`, `history-rm <id>`, `history-clear`,
  `count`.
- dunst ≥ 1.8 supports `#RRGGBBAA` colors on Wayland → translucent toasts are
  possible. dunst's layer surface uses the **`notifications`** namespace
  (verified via `hyprctl layers` with a toast up), so Hyprland can blur behind
  it.
- The current dunst config logs startup warnings (invalid `icon_size` global
  key, legacy `offset = "16x56"` and `height` syntax); the restyle fixes these
  while rewriting the settings.
- `SUPER+N`, `SUPER+CTRL+N`, and `SUPER+ESCAPE` are unbound in
  `hyprland/default.nix`. `ctrl+space` must stay untouched (fcitx5 IME
  toggle).

## Design Language: Luminous Glass (G3)

Chosen via visual mockups (frosted glass direction, "luminous" flavor):
glass surfaces + a whisper of per-module neon.

**Palette source of truth.** New file
`modules/desktop/home/hyprland/theme/palette.nix`: a plain attrset of
Catppuccin Mocha hex values (no option plumbing; consumers `import` it by
relative path). Consumers:

- `eww/default.nix` renders it to a `_palette.scss` partial (via
  `pkgs.writeText`) that `eww.scss` imports, replacing the inline variable
  block.
- `dunst.nix` references the attrset instead of hardcoded hex strings.
- `hyprlock.nix` is left alone for now (out of scope), but can adopt it later.

The palette stays **static Catppuccin Mocha**. Wallpaper-adaptive colors were
considered and deferred to the wallpaper sub-project; extracting the palette
into one file is the enabling refactor either way.

**Surfaces.** Bar islands and all popup cards become translucent —
`rgba($crust, ~0.60)` — with Hyprland blur behind them. The single
`eww-bar` blur layerrule is replaced by a regex rule covering all eww
namespaces (`match:namespace eww-.*, blur on`) plus one for `notifications`
(dunst's layer namespace).

**Accent borders + glow.** Each island/popup keeps its existing accent color
but expresses it as: 1px border tinted with the accent at ~35–45% alpha, plus
a faint outer glow (`box-shadow: 0 0 ~10px accent@~15%`) layered under the
existing soft drop shadow. Text accents stay as today. The active workspace
dot gets a small glow in its accent. Exact alphas/radii are calibration
values, tuned live at the end (same process as the popup calibration pass).

**Toasts.** dunst adopts the same language: translucent base
(`$crust` at ~0x99 alpha), accent frame per urgency (mauve for normal, red
for critical), corner radius and typography matched to the popup cards. The
rewrite also clears dunst's startup warnings by dropping the invalid
`icon_size` key and migrating `offset`/`height` to the current syntax.

## Notification Pipeline (backend)

New collector `notifications_state()` in `collectors.py`, following the
existing dict-state pattern:

```
{
  "paused": false,
  "new": 2,            # items newer than last mark-seen
  "count": 7,          # total items in history
  "groups": [
    {"app": "Element", "count": 2, "collapsed": false,
     "items": [{"id": 4, "summary": "...", "body": "...",
                "age": "2m", "urgency": "NORMAL"}]},
    ...
  ]
}
```

- Groups are keyed by `appname` (fallback `"unknown"`), ordered by their
  newest item, items newest-first. Truncation guards popup width: app ~20,
  summary ~48, body ~64 chars (reuse `truncate_text`).
- `age` is computed from the monotonic timestamp against `/proc/uptime` and
  formatted `now / Ns / Nm / Nh / Nd`.
- `paused` comes from `dunstctl is-paused`.
- **UI state lives in the backend** (same pattern as display mode): the set of
  collapsed app groups and the `last_seen` monotonic timestamp that defines
  `new`. This state is process-lifetime only; it resets with the daemon
  (acceptable: collapse state and badges are ephemeral by nature).

**Refresh triggers.** A `dbus-monitor` (or `busctl monitor`) thread watches
`org.freedesktop.Notifications` for `Notify` / `NotificationClosed` traffic
and schedules a debounced re-read. If the monitor process dies it is
respawned; as a safety net the collector also re-reads on a slow poll.
Every control verb below triggers an immediate re-read after acting.

**Control verbs** (new `notif` group in `control.py`, dispatched over the
existing socket):

| verb | effect |
| --- | --- |
| `notif toggle-group <app>` | flip collapse state for that group |
| `notif dismiss <id>` | `dunstctl history-rm <id>` |
| `notif clear-group <app>` | `history-rm` every id in that group |
| `notif clear-all` | `dunstctl history-clear` |
| `notif dnd-toggle` | `dunstctl set-paused toggle` |
| `notif mark-seen` | set `last_seen` = newest item's timestamp |

## Notification Center Popup

New `defwindow notif_center_popup` (`:namespace "eww-notifications"`),
registered in `POPUP_WINDOWS` so it inherits the whole popup lifecycle:
backdrop, click-outside close, mutual exclusion with other popups, consistent
top-right corner (`x=12px, y=50px`), `:timeout "5s"` on triggers.

Layout (grouped-by-app, chosen via mockups):

- **Header** (shared `popup_header`, mauve accent): bell mark + "Notifications"
  title, a DND pill toggle (shows state, calls `notif dnd-toggle`), and a
  clear-all button.
- **Groups**: header row = app name + count badge; clicking it toggles a
  `revealer` (calls `notif toggle-group`). A small ✕ on the group header
  clears the group.
- **Items**: summary (bold) + body (subtext), age right-aligned, per-item ✕
  (`notif dismiss`). Critical-urgency items get a red accent tint.
- **Empty state**: centered "No notifications".
- The list lives in a `scroll` widget with a max height (~480px) so long
  histories scroll instead of growing the window.

`history_length` is raised 20 → 50 in dunst, since history is now visible.

## Bar Bell Module

New small button-island in the right cluster (initial placement: between the
tray island and the network island; final position is a calibration decision).

- Default: bell glyph, dimmed when `new == 0`; shows `new` count when > 0.
- DND on: crossed-bell/moon glyph replaces the bell (badge hidden).
- Click: `eww-popup toggle notif_center_popup` **and** `notif mark-seen`
  (opening the center resets the badge).

## Keybinds (Hyprland)

Replace dunst's dead X11 shortcuts (removed from `dunst.nix`):

- `SUPER+N` — toggle notification center (`eww-popup toggle notif_center_popup`,
  plus `mark-seen`)
- `SUPER+CTRL+N` — `eww-barctl --quiet notif dnd-toggle`
- `SUPER+ESCAPE` — `dunstctl close` (dismiss current toast)
- `SUPER+SHIFT+ESCAPE` — `dunstctl close-all`

The eww-dependent binds are guarded by the eww enable option, like the
existing `eww-barctl display toggle` bind.

## Error Handling

- All `dunstctl` calls go through the existing timeout-guarded `run_text`
  helper; failures or malformed JSON yield `NOTIFICATIONS_DEFAULT` (empty
  groups, `paused: false`) — the bar never breaks because dunst is down.
- Unknown ids/apps in control verbs are no-ops (dunstctl already tolerates
  them; `check=False` pattern).
- If the dbus-monitor thread dies, it respawns with backoff; the slow poll
  keeps state eventually-correct meanwhile.

## Testing

- Unit tests (pure functions, fixture-driven, in `tests/eww_bar_backend/`):
  history-JSON → grouped state (grouping, ordering, truncation, urgency),
  age formatting boundaries, monotonic-age math, `new`-count vs `last_seen`,
  clear-group id selection, paused parsing, verb routing.
- `python3 -m py_compile` on touched modules; `eww` config compile check.
- Live verification (controller, after user-triggered rebuild): open/close via
  bell + `SUPER+N`, DND toggle from popup and keybind, dismiss/clear flows,
  `notify-send` smoke tests at all urgencies, grim screenshots for the glass
  restyle on bar/popups/toasts.

## Out of Scope

- Rofi / wlogout / hyprlock restyle (later polish pass).
- Wallpaper-adaptive palette generation (wallpaper sub-project; enabled by
  `palette.nix` but not built here).
- Invoking notification actions from history (`dunstctl` cannot trigger
  actions on dismissed notifications).
- Per-app notification rules/filters.
