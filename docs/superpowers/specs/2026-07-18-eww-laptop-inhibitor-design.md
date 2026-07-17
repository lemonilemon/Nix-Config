# Eww Laptop Inhibitor Design

## Goal

Replace the current Eww idle inhibitor with a more reliable control surface that pauses
Hypridle only, then add laptop-only display presets for clamshell and temporary
headless/server use.

## Current State

The Eww bar currently exposes an idle inhibitor button that calls
`eww-barctl --quiet idle toggle`. The backend starts `systemd-inhibit --what=idle`
with `sleep infinity`, writes the child PID to `$XDG_RUNTIME_DIR/eww-idle-inhibit.pid`,
and treats the PID as the source of truth.

That works in the common case, but it is brittle:

- PID reuse can make the bar report a stale inhibitor as active.
- Turning the inhibitor off can signal the wrong process if the PID was reused.
- The inhibitor is not managed as a user service, so lifecycle and debugging are weak.
- The older `scripts/idle-toggle` shell implementation still exists beside the Python
  backend path and can drift.

The laptop profile does not currently configure lid-switch behavior. Hyprland monitor
layout is delegated to `~/.config/hypr/monitors.conf`, with a generic fallback
`monitor=,preferred,auto,1`.

## User-Facing Behavior

The Eww inhibitor button will become a compact "stay awake / display mode" control.

Left click:

- Toggle a Hypridle-only pause.
- This must not block explicit suspend actions such as `systemctl suspend` or wlogout
  suspend.

Right click:

- Open a small Eww popup for laptop display behavior.
- The popup is laptop-only. Desktop keeps the simple inhibitor button.

Popup actions:

- `Pause Hypridle`: toggles the same Hypridle-only pause as left click.
- `Exit mode`: restore normal monitor behavior and clear laptop-specific display mode.
- `External only`: keep the system running with the lid closed, keep external monitors
  active, and disable the internal laptop panel.
- `Headless server`: keep the system running with the lid closed and turn displays off
  temporarily.
- `Turn screens on`: turn displays back on without clearing the current display mode or
  inhibitors.

The `External only` and `Headless server` presets should also pause Hypridle. Otherwise
the existing lock, DPMS, or suspend listeners could still fire while the machine is
being intentionally kept awake.

## Architecture

Use systemd user services as the runtime source of truth instead of PID files.

Planned services:

- `eww-hypridle-inhibit.service`
  - Runs `systemd-inhibit --what=idle --who=eww-bar --why=User paused Hypridle sleep infinity`.
  - Pauses Hypridle listeners because Hypridle honors systemd idle inhibitors by default.
  - Does not inhibit explicit sleep requests.

- `eww-lid-inhibit.service`
  - Runs `systemd-inhibit --what=handle-lid-switch --who=eww-bar --why=Laptop display mode keeps lid close ignored sleep infinity`.
  - Used only on laptop display presets that require lid-close behavior to be ignored.
  - Keeps normal lid behavior unchanged when no preset is active.

The Eww backend will call `systemctl --user start/stop/is-active` for these services.
State emitted to Eww will come from systemd service state, not `/proc/<pid>`.

## Backend Commands

Extend `eww-barctl` with display commands:

- `eww-barctl idle toggle|on|off|status`
- `eww-barctl display normal|external|headless|restore|status`

State additions:

- `idle_inhibited`: string `"true"` or `"false"`.
- `lid_inhibited`: string `"true"` or `"false"`.
- `display_mode`: string such as `"normal"`, `"external"`, or `"headless"`.
- Optional short tooltip/status text for the popup.

Mode state can be stored in `$XDG_RUNTIME_DIR/eww-display-mode`. It is runtime-only on
purpose, so modes do not persist across reboot.

## Monitor Behavior

The backend will use `hyprctl` for display actions.

Internal panel detection:

- Treat monitor names beginning with `eDP-` or `LVDS-` as internal laptop panels.
- Treat all other monitors as external.

`External only`:

- Start Hypridle inhibitor.
- Start lid-switch inhibitor.
- Disable the internal panel through Hyprland monitor commands.
- Keep external monitors active.
- If no external monitor is detected, refuse the mode and leave the current display
  setup unchanged to avoid intentionally blanking the only visible display.

`Headless server`:

- Start Hypridle inhibitor.
- Start lid-switch inhibitor.
- Use `hyprctl dispatch dpms off` to power off displays temporarily.
- Prefer DPMS over permanently rewriting monitor layout, because recovery is simpler.

`Turn screens on`:

- Run `hyprctl dispatch dpms on`.
- Keep the current display mode.
- Keep the current Hypridle and lid-switch inhibitor states.

`Exit mode`:

- Run `hyprctl dispatch dpms on`.
- Reload/apply normal monitor configuration.
- Stop the lid-switch inhibitor.
- Stop the Hypridle inhibitor.
- Clear display mode back to `normal`.

## Recovery

Because `Headless server` can make the machine invisible locally, recovery cannot depend
only on the popup.

Add recovery paths:

- Eww popup `Turn screens on` action.
- CLI fallback: `eww-barctl display restore`.
- Laptop-only Hyprland keybind, for example `SUPER+SHIFT+O`, that runs the restore
  command even when all displays are off.

## Nix Configuration

Add laptop-only Home Manager options under the existing Eww module shape:

- `home.desktop.hyprland.eww.laptopControls.enable`

Defaults:

- `false` globally.
- Enabled in `profiles/laptop/config.nix`.

Desktop behavior remains unchanged except for the improved Hypridle inhibitor backend.

The systemd user services belong in `modules/desktop/home/hyprland/eww/default.nix`
because they are part of the Eww control surface and use the same runtime package path.

## UI

Keep the existing button compact in the tray island.

- Active Hypridle pause keeps the current active styling.
- Right-click opens a small popup near the bar.
- Popup rows are direct controls, not documentation text.
- Laptop-only controls should not render on desktop.

## Testing And Verification

Local checks:

- `python3 -m py_compile` for the Eww backend package.
- Targeted backend tests with mocked `systemctl`, `systemd-inhibit`, and `hyprctl` if
  practical.
- `just fmt`.
- `NIXHOST=laptop just test`.
- `NIXHOST=desktop just test`.
- `NIXHOST=laptop just dry-build` if evaluation passes and sudo is appropriate for the
  user's workflow.

Manual checks after rebuild:

- Left-click toggles Hypridle pause.
- `systemctl --user is-active eww-hypridle-inhibit.service` matches Eww state.
- Right-click popup appears on laptop.
- `External only` refuses to run without an external monitor.
- `Headless server` can be recovered with the keybind or CLI restore command.
- `Turn screens on` restores visibility without leaving the current mode.
- `Exit mode` restores visibility and clears the display-mode inhibitors.
- Desktop does not show laptop-only popup controls.
