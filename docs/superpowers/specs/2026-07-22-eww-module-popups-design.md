# Eww Module Popups Design

## Goal

Replace the "launch an external app / show a plain tooltip" behavior of four
right-side bar modules with self-contained Eww popups, and unify **every** popup
(the two existing ones included) under a single interaction model:

- **Open** with a left click that toggles the popup.
- **Position** the popup under the module that triggered it.
- **Close** on a click anywhere outside the popup, as well as on a second click
  of the module.

The four modules are **Volume/Audio**, **Bluetooth**, **Network**, and
**Battery**. CPU/memory keep launching `htop` (a terminal process viewer is the
right tool there) and are out of scope.

## Current State

The bar (`modules/desktop/home/hyprland/eww/eww.yuck`) drives all modules from a
single `deflisten bar_state` fed by the Python backend
(`scripts/eww_bar_backend/`, exposed as `eww-bar-backend` and the `eww-barctl`
control alias). Two popups already exist and share a card design language
(`rgba(17,17,27,0.985)` background, `12px` radius, hairline border, large soft
shadow, header = colored "mark" icon + title stack):

- `ai_usage_popup` — opened by left-clicking the AI-usage module; also fires
  `eww-barctl --quiet ai refresh`.
- `display_mode_popup` — opened by right-clicking the idle inhibitor
  (laptop-only), closed from an in-popup "Exit mode" button.

Both open with a raw `eww open --toggle <name> --screen ${output}` and sit at a
fixed top-right offset. There is **no click-outside-to-close**: `:focusable
false` popups get no unfocus event, so a popup left open stays open until its
module (or button) is clicked again.

Today's target modules behave as follows:

- **Volume** — scroll adjusts the level in place; left click launches
  `kitty -- pulsemixer`. State is a bare string (`" 45%"` or `"󰝟"`).
- **Bluetooth** — a `label` with a hover tooltip listing connected devices. State
  is already a dict (`text`/`tooltip`/`class`).
- **Network** — a `label` with a hover tooltip showing device + IP. State is a
  dict (`text`/`tooltip`/`class`).
- **Battery** — left click toggles the bar text between charge % and time
  remaining. State is a dict (`text`/`alt`/`capacity`/`class`).

Environment facts that constrain the design (verified in the repo/host):

- eww is `0.6.0` — `scale` sliders (`:onchange`), `revealer` transitions, and
  `circular-progress` are available.
- Two monitors: `eDP-1` (internal) and `HDMI-A-1` (external). Bars open
  per-monitor; popups open on the clicked bar's screen via `--screen ${output}`.
- `power-profiles-daemon` is **disabled**; the host uses `auto-cpufreq`
  (automatic). There is no simple profile switch to expose, so the Battery popup
  is **detail-only** (no controls).
- `blueman` is enabled → the Bluetooth popup can launch `blueman-manager`.
- No NetworkManager GUI is installed, but `nmtui` ships with NetworkManager → the
  Network popup's advanced action opens `kitty -- nmtui`.
- `wpctl`/`pactl` (audio) and `bluetoothctl`/`nmcli` are already on the eww
  runtime `PATH`.

## Shared Interaction Model

### Open

Left-clicking a module toggles its popup. Scroll on the volume module keeps
adjusting the level in place with no popup, exactly as today.

### Position — under each module

Each popup is a layer-shell window anchored **top-right** with a **per-module
horizontal offset** (px from the right screen edge) so it sits roughly beneath
its own icon, dropping just below the bar (`y ≈ 50px`).

The whole `.right` section is right-aligned (`:halign "end"`), so measuring the
offset from the **right edge** makes the alignment **independent of monitor
width** — the same offsets work on `eDP-1` and `HDMI-A-1`. The only case that
shifts a neighbor is the Bluetooth module hiding when nothing is connected;
that is an accepted minor drift. Exact offsets are calibrated against a real
screenshot (`grim`) during implementation. Right-to-left module order (and thus
increasing offset) is: power, **battery**, **volume**, memory, cpu, temperature,
ai-usage, **bluetooth**, **network**, idle-inhibitor, tray.

### Close — a shared backdrop below the bar

A single transparent window, `popup_backdrop`, covers the screen **below the
bar** (anchored top, `y` at the bar's bottom edge) and runs `eww-popup close` on
any click. Anchoring below the bar (rather than full-screen) keeps every gesture
a single click:

```
click module (popup closed)     -> open backdrop + popup
click the same module again     -> module isn't covered -> helper closes it
click a different module        -> helper closes the old popup and opens the new one
click anywhere in the app area  -> backdrop catches it -> close
```

Trade-off: a small dead zone over *empty* bar space (never over a module) does
not dismiss. This is acceptable and rarely hit.

Stacking: the helper opens `popup_backdrop` first and the popup second, so the
popup renders above the backdrop and stays clickable while the backdrop catches
everything else. All popups and the backdrop use `:stacking "fg"`.

### Orchestration helper: `eww-popup`

Popup lifecycle is centralized in a small helper so the logic is testable and
multi-monitor-correct. It is added to the backend package
(`eww_bar_backend/popups.py`) and installed as a third entry point next to
`eww-bar-backend`/`eww-barctl` (dispatched by `argv[0]`), so it lives on the same
runtime `PATH`.

Interface:

- `eww-popup toggle <window-name> <screen>`
- `eww-popup close`

Behavior (pure decision logic, with `eww active-windows` and `eww` calls injected
for testing):

1. Read `eww active-windows` (format `id: window-name`; a popup opened without
   `--id` has `id == name`).
2. Always `eww close <all popup names> popup_backdrop` first (single popup at a
   time; also clears a stale backdrop).
3. If the requested popup was **not** already open, `eww open popup_backdrop
   --screen <screen>` then `eww open <window-name> --screen <screen>`. If it was
   open, step 2 already closed it (toggle-off).

Because popups and the backdrop are single-instance (default id = name),
re-opening with a different `--screen` moves them to the clicked monitor.

The two existing popups are retrofitted onto this helper: the AI-usage module's
onclick becomes `eww-popup toggle ai_usage_popup ${output} && eww-barctl --quiet
ai refresh`; the idle inhibitor's right click becomes `eww-popup toggle
display_mode_popup ${output}`; and their in-popup close buttons call `eww-popup
close`.

## The Popups

All four reuse the established card design language, implemented as shared
`.popup-*` SCSS classes (base card, header, mark, title/eyebrow, rows, actions,
slider). Each popup carries an accent color matching its bar module: volume =
yellow, bluetooth = sapphire, network = peach, battery = blue. Existing
`.ai-usage-popup` / `.display-mode-popup` styles are left as-is.

### Volume (accent yellow)

- A `scale` slider bound to the live level, `:onchange "eww-barctl --quiet volume
  set {}"`.
- A mute toggle button (active styling when muted), `eww-barctl --quiet volume
  mute`.
- One row per output sink; clicking a row makes it the default sink
  (`eww-barctl --quiet volume sink <name>`). The active sink is marked.

```
┌───────────────────────────────┐
│ []  Volume              45%   │   header mark + live %
│ ━━━━━━━━●────────────────────  │   scale -> volume set {}
│ []  Mute                      │   toggle, active when muted
│ ─────────────────────────────  │
│ Output                        │
│ ● Built-in Analog     (active) │   sink rows, click to switch
│ ○ UGREEN USB-C                │
└───────────────────────────────┘
```

Note: `scale :onchange` fires repeatedly while dragging; each event spawns
`eww-barctl` → backend socket → `wpctl`. This is acceptably cheap and needs no
throttling.

### Bluetooth (accent sapphire)

- Power on/off toggle (`eww-barctl --quiet bluetooth power-toggle`).
- One row per connected device: name + battery %, with a disconnect action
  (`eww-barctl --quiet bluetooth disconnect <mac>`).
- An "Open blueman" action → `hyprctl dispatch exec blueman-manager`.
- Pairing/connecting new devices is delegated to blueman (out of scope here).

### Network (accent peach)

- A connection card: Wi-Fi SSID + signal % + IP, or ethernet device + IP, or a
  disconnected state.
- A Wi-Fi on/off toggle (`eww-barctl --quiet network wifi-toggle`).
- An "Advanced…" action → `hyprctl dispatch exec 'kitty -- nmtui'`.
- Scanning/joining arbitrary networks is delegated to nmtui (out of scope here).

### Battery (accent blue) — detail only

- Charge %, charging state, and time-to-full/empty.
- Battery **health** (`energy_full / energy_full_design`).
- Instantaneous **power draw** in watts.
- No controls (auto-cpufreq manages scaling; nothing to switch).

## Backend State and Commands

### Collectors (`collectors.py`)

- **Volume** becomes a dict:
  `{text, percent, muted, icon, class, sinks:[{name, description, active}]}`.
  - `percent` (int 0–100) drives the slider; `text` keeps the bar label (icon +
    percent, or `"󰝟"` when muted); `muted`/`class` drive styling.
  - Sinks parsed from `pactl -f json list sinks` (technical `name` +
    human-readable `description`) with the default from `pactl get-default-sink`;
    fall back to `pactl list short sinks` if JSON is unavailable. Rows display
    `description`; clicking sends the technical `name` to `volume sink <name>`.
- **Bluetooth** dict gains `powered` (`"true"/"false"`) and
  `devices:[{mac, name, battery, connected}]`, built from the same
  `bluetoothctl show` / `devices Connected` / `info <mac>` data already
  collected. Existing `text`/`tooltip`/`class` unchanged.
- **Network** dict gains `{kind, ssid, ip, signal, wifi_enabled, device}`
  alongside existing `text`/`tooltip`/`class`. `wifi_enabled` from
  `nmcli radio wifi`.
- **Battery** dict gains `status`, `health`, `power`, and an explicit `time`
  (the value currently only embedded in `alt`).

### Control verbs (`control.py`)

Extend the control command surface (and `CONTROL_USAGE`):

- `volume up|down` (existing) plus `volume set <0-100>`, `volume mute`,
  `volume sink <name>`.
  - `set` → `wpctl set-volume @DEFAULT_AUDIO_SINK@ <n>%`.
  - `mute` → `wpctl set-mute @DEFAULT_AUDIO_SINK@ toggle`.
  - `sink <name>` → `pactl set-default-sink <name>`.
  - Each returns the fresh `volume_state()` and updates `state`.
- `bluetooth power-toggle`, `bluetooth disconnect <mac>` (via `bluetoothctl`),
  each returning fresh `bluetooth_state()`.
- `network wifi-toggle` (via `nmcli radio wifi on|off`), returning fresh
  `network_state()`.

`control_payload_from_args` gains parsing for the new argument shapes.

### Defaults and the deflisten seed

The volume/bluetooth/network/battery shapes are updated in all three places that
hold initial state so the bar renders before the first collector runs:

- `common.py` — add a `VOLUME_DEFAULT` dict; extend `BATTERY_DEFAULT`; provide
  explicit bluetooth/network defaults (replacing bare `EMPTY_MODULE`).
- `state.py` — `BarState.__init__` initial dict.
- `eww.yuck` — the `deflisten bar_state :initial '{...}'` JSON seed.

The volume bar label in `eww.yuck` changes from `{bar_state.volume}` to
`{bar_state.volume.text}`.

## Widgets and Styling

### `eww.yuck`

- New windows: `volume_popup`, `bluetooth_popup`, `network_popup`,
  `battery_popup`, and `popup_backdrop`.
- New widgets: `volume_panel`, `bluetooth_panel`, `network_panel`,
  `battery_panel`, plus small shared helpers (e.g. a `popup_header` widget and a
  `popup_action` button) to avoid duplication.
- Module wiring on the bar:
  - Volume: keep the `eventbox :onscroll`, change the inner button's onclick to
    `eww-popup toggle volume_popup ${output}`; label → `.text`.
  - Bluetooth / Network: convert the `label` to a `button` with onclick
    `eww-popup toggle <name> ${output}` (preserve visibility conditions).
  - Battery: replace the `show_battery_time` toggle onclick with
    `eww-popup toggle battery_popup ${output}`.
  - AI-usage / idle-inhibitor: route through `eww-popup` as described above.
- Device/sink lists render with static indexing or bounded `for` loops
  consistent with the existing quota-card note about eww re-appending `for`
  children on array updates; connected-device and sink counts are small and
  variable, so a `for` with a stable parent is acceptable here (unlike the
  fixed-provider quota cards). This will be validated during implementation; if
  blanking recurs, fall back to a small fixed set of indexed rows.

### `eww.scss`

- Add shared `.popup`, `.popup-header`, `.popup-mark`, `.popup-title`,
  `.popup-eyebrow`, `.popup-body`, `.popup-row`, `.popup-action`,
  `.popup-slider`, `.popup-divider` classes derived from the existing card
  values, with `.accent-yellow|sapphire|peach|blue` (or per-popup) modifiers.
- Add `.popup-backdrop` as fully transparent (`background: transparent`) — it
  only needs to catch clicks. (A subtle dim is a trivial future tweak.)
- Leave existing popup styles untouched.

## Nix Configuration (`eww/default.nix`)

- Add `cp $out/bin/eww-bar-backend $out/bin/eww-popup` to the `ewwBarTools`
  `installPhase` (alongside the existing `eww-barctl` copy).
- Add `blueman` to `runtimePackages` (for `blueman-manager`). `wpctl`
  (wireplumber), `pactl`/`pulseaudio`, `bluetoothctl` (bluez), `nmcli`
  (networkmanager, which also provides `nmtui`), and `kitty` are already present.
- No new privileges or services are required.

## Testing and Verification

Local checks:

- `python3 -m py_compile` over the backend package (including `popups.py`).
- Backend `unittest` additions under `tests/eww_bar_backend/`, run with
  `cd tests/eww_bar_backend && python3 -m unittest discover -p 'test_*.py'`
  (root-level `discover` collides with the real package name). New coverage:
  - Volume parsing: percent, muted, icon, sink list + active flag (JSON and
    short-listing fallback).
  - Bluetooth device-list parsing and `powered`.
  - Network structured fields incl. `wifi_enabled`.
  - Battery `health`/`power`/`status` derivation.
  - Control routing for `volume set|mute|sink`, `bluetooth power-toggle|
    disconnect`, `network wifi-toggle` (subprocess mocked, asserting exact
    argv).
  - `eww-popup` decision logic: toggle-open, toggle-close, switch, and `close`,
    fed a fake `active-windows` string and a fake `eww` runner that records the
    exact call sequence and ordering (backdrop before popup).
- `just fmt` (`nix fmt`).
- `NIXHOST=laptop just test` (eval) and `NIXHOST=laptop just dry-build`.

Manual checks after `NIXHOST=laptop just build` and
`systemctl --user restart eww-bar`:

- Each module's left click opens its popup under the module on the clicked
  monitor; a `grim` screenshot confirms/calibrates the offsets.
- Clicking the same module, a different module, or empty desktop closes/switches
  correctly.
- Volume slider changes the level; mute toggles; switching sinks works.
- Bluetooth power toggle + disconnect + "Open blueman" work; Network Wi-Fi
  toggle + "Advanced…" work.
- Battery popup shows charge/status/time/health/power.
- The retrofitted AI-usage and display-mode popups still open and now
  close on outside click.
- Popups open on the correct screen on both `eDP-1` and `HDMI-A-1`.

## Build Order

Each step is independently shippable and verifiable:

1. `popup_backdrop` window + `eww-popup` helper + retrofit the two existing
   popups — proves the whole open/position/close mechanism end to end.
2. **Volume** popup — establishes the slider + sink-row pattern and the volume
   state/control changes.
3. **Bluetooth** popup.
4. **Network** popup.
5. **Battery** popup.

## Non-Goals

- No power-profile / cpufreq switching (PPD disabled; auto-cpufreq is automatic).
- No brightness control in this iteration.
- No in-popup Wi-Fi scanning/joining or Bluetooth pairing (delegated to nmtui /
  blueman).
- No CPU/memory popup (htop remains the drill-down).
- No animated reveal transitions in v1 (the `revealer` option can be added later
  without changing the interaction model).
