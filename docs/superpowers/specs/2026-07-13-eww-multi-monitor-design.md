# Eww Multi-Monitor Lifecycle Design

## Goal

Make the Eww bar behave like the previous Waybar setup: display one bar on every active Hyprland output at login and automatically add or remove bars when the monitor topology changes.

## Current Behavior and Root Cause

The Eww user service starts the daemon and runs a one-time `ExecStartPost` script. That script queries `hyprctl monitors -j` and opens one uniquely identified bar per output. This supports monitors present during startup, but it has no persistent owner that reacts to later topology changes.

Eww notices a newly connected display and reloads its configuration. During that reload it restores existing windows, but it does not create a new window instance for the new output. The journal confirms this behavior: after a monitor-connect event, Eww reopened only the existing `bar-eDP-1` instance.

## Chosen Approach

Add an event-driven monitor manager as a companion systemd user service. The manager will:

1. Wait until the Eww daemon is reachable.
2. Query the active output names from `hyprctl monitors -j`.
3. Reconcile managed Eww windows so the active set contains exactly one `bar-<output>` instance per output.
4. Listen to Hyprland's socket2 event stream.
5. Debounce `monitoradded`, `monitoraddedv2`, `monitorremoved`, and `monitorremovedv2` events before reconciling again, allowing Hyprland and Eww's own display reload to settle.

This is preferred over polling because it reacts immediately without periodic wakeups. It is preferred over restarting the Eww daemon because it avoids resetting unrelated Eww windows, variables, and backend state.

## Components

### Monitor manager script

A focused shell script will own bar lifecycle. It will support a one-shot reconciliation path for startup and testing, followed by a persistent event-listening path in normal operation.

The script will keep the previous output set in memory. During reconciliation it will close bar IDs for the union of the previous and current output sets, then reopen one bar for each current output using:

```text
eww open bar --id bar-<output> --screen <output> --arg output=<output>
```

Closing only these stable IDs ensures the AI usage popup and any future non-bar Eww windows remain untouched.

If no output is available during a transient reconfiguration, the manager will leave no bar open and retry on the next monitor event. Failure to contact Eww or query Hyprland will cause the manager service to fail so systemd can restart it rather than silently leaving stale state.

### systemd user services

The existing `eww-bar` service will remain responsible only for the Eww daemon. Its one-time bar-opening `ExecStartPost` action will be removed.

A new companion service will:

- start after and require `eww-bar.service`;
- remain part of `hyprland-session.target`;
- run the monitor manager as its main process;
- restart on failure;
- receive the same runtime command path needed for `eww`, `hyprctl`, `jq`, and `socat`.

Keeping daemon and lifecycle responsibilities separate makes failures and logs attributable to the correct component.

## Event and Data Flow

```text
Hyprland session starts
  -> Eww daemon starts
  -> monitor manager starts
  -> manager enumerates outputs
  -> manager opens one targeted bar per output

Hyprland monitor event
  -> manager debounces the event
  -> manager re-enumerates outputs
  -> manager closes its previous/current bar IDs
  -> manager opens the exact current bar set
```

## Testing and Verification

A shell-level regression test will place mocked `hyprctl`, `eww`, and event input ahead of the real commands in `PATH`. It will verify:

- two active outputs produce two unique `eww open` commands;
- each command carries the correct `--id`, `--screen`, and `output` argument;
- a removed output's stable bar ID is closed;
- unrelated Eww window IDs are never closed.

Repository verification will then run `just fmt`, `just test`, and `just dry-build` as required by the NixOS configuration workflow. The live system will not be switched automatically; the user will apply the verified configuration with `just build`.

## Scope

This change only manages Eww bar instances across monitor topology changes. It does not change bar layout, workspace filtering, popup design, or the shared backend state.
