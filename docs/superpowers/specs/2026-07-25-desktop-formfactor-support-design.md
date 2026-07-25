# Desktop Form-Factor Support Design

## Goal

Make "this host is a laptop / a desktop / WSL" a single declared fact that every
laptop-vs-desktop difference derives from, instead of the three parallel
mechanisms in use today (which profile fragments a host imports, which power
settings it inlines, and a hand-set `home.desktop.hyprland.eww.laptopControls.enable`).

Four domains come under that fact: power and thermal, networking posture, the
Eww bar, and idle/lid/sleep policy.

## Current State

### Profiles diverge by omission

`profiles/laptop/default.nix` imports `../network.nix`, `../firewall.nix` and
`../smb.nix`. `profiles/desktop/default.nix` imports none of them and instead
inlines a partial copy of the NetworkManager block (no `lib.mkDefault`, no
`networking.wireless.enable`, no `hardware.enableRedistributableFirmware`). The
desktop therefore has **no firewall configuration of its own**, and its NM
settings cannot be overridden by a later module because they are not
`mkDefault`.

Corrected after implementation (2026-07-25): "no firewall configuration" is not
"no firewall". `networking.firewall.enable` defaults to `true` in NixOS, so the
desktop is firewalled today with **zero** open ports. Adopting the shared list
therefore *opens* 22/80/443, and `trustedSubnets` adds an all-ports accept for
the whole LAN. That is a loosening, not a hardening, and the decision below
should be read in that light.

The same correction applies to the laptop. `profiles/firewall.nix` writes
`allowedTCPPorts = lib.mkDefault [ 22 80 443 ]`, and because `types.listOf`
filters overrides before concatenating, Samba's normal-priority `[ 139 445 ]`
discarded that list wholesale. The laptop never had 22/80/443 open either;
after this work it does.

`profiles/firewall.nix` hardcodes:

```
nft add rule inet filter input ip saddr 192.168.1.0/24 accept
```

This machine is on `192.168.0.0/24`, so the rule matches nothing. It is also
issued through `networking.firewall.extraCommands`, which the iptables backend
(the NixOS default) executes as iptables rules — so the `nft` invocation is
unlikely to have applied cleanly on either host. The implementation must
generate the rule through the backend-appropriate mechanism and confirm the
firewall unit starts clean.

### Power settings are laptop-shaped on a workstation

`profiles/desktop/default.nix` sets `powerManagement.powertop.enable = true` and
enables `auto-cpufreq` alongside `powerManagement.cpuFreqGovernor = "performance"`,
on a wall-powered Intel i7-9700KF. auto-cpufreq rewrites the governor at
runtime, so the static setting is decorative — two things claim ownership of the
same knob. `services.upower.enable` is laptop-only, which is correct but is
stated by omission rather than derived.

### The bar cannot tell "NetworkManager is dead" from "offline"

`network_connection_state()` (`modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py:1036`)
shells out to `nmcli` exclusively. `run_text()` (`common.py:89`) swallows every
failure into `""`. Empty output matches no wifi device and no ethernet device,
so control reaches the fall-through at `collectors.py:1081` and returns
`⚠ Disconnected`.

Observed on 2026-07-25: `sudo nixos-rebuild switch --flake .#desktop` at
19:12:51 left NetworkManager stopped at 19:13:12 (along with nscd, thermald,
auto-cpufreq, powertop, cpufreq, systemd-oomd, resolvconf,
network-local-commands and systemd-modules-load). `eno1` kept its lease —
`192.168.0.88/24` with a default route via `192.168.0.1` — so the machine stayed
online while the bar read `⚠ Disconnected`. Running the collector directly
confirmed it:

```
{'text': '⚠ Disconnected', 'tooltip': 'No connection', 'class': 'disconnected', 'wifi_enabled': 'false'}
```

The bar is also wifi-first: `network_connection_state()` checks for a connected
wifi device before ethernet, so on a host with both, wifi wins even when the
cable carries the default route.

### Idle policy is hardcoded

`services.hypridle` at `modules/desktop/home/hyprland/hyprlock.nix:110` fixes
lock at 15 min, DPMS off at 20 min and `systemctl suspend` at 30 min for every
host that enables hyprland.

## Design

### The `formFactor` option

Declared once, at top level, because it describes the machine rather than one
subtree.

Changed during implementation: it lives in `modules/nixos.nix`, **not** the
shared `modules/options.nix`. That file is imported by both `modules/nixos.nix`
and `modules/home.nix`, so declaring it there gives the Home Manager tree its
own copy that mirrors nothing and always reads `"desktop"` — an option that
evaluates fine while being silently wrong on the laptop. Keeping it out of the
HM tree turns that trap into a hard eval error, and forces HM values to arrive
by mirroring their NixOS twin through `helpers.mkHomeOpt`.

```nix
formFactor = lib.mkOption {
  type = lib.types.enum [ "laptop" "desktop" "wsl" ];
  default = "desktop";
  description = "Physical form factor of this host; seeds power, network, idle and bar defaults.";
};
```

Each profile sets exactly one line: `formFactor = "laptop" | "desktop" | "wsl"`.

### Derived capability options

Every difference is a named option whose *default* is an expression over
`config.formFactor`, in the cascading style the repo already uses
(`nixos.general.nixld.enable` defaulting from `nixos.general.enable`):

```nix
nixos.general.power.governor = lib.mkOption {
  type = lib.types.str;
  default = if config.formFactor == "laptop" then "powersave" else "performance";
  description = "CPU frequency governor for this host.";
};
```

An unusual host stays expressible: set `formFactor = "desktop"` and still
override a single leaf, without editing a module.

### Invariant

**Module `config` blocks never read `formFactor`. They read capability options
only. Only `options.nix` files mention `formFactor`.**

This is what keeps the mechanism from degenerating back into scattered
conditionals, and it is the rule to check when adding a host or a domain.

### Home Manager mirroring

No new machinery. The NixOS-side option (for example
`home.desktop.hyprland.eww.laptopControls.enable` in `modules/desktop/options.nix`)
takes its default from `config.formFactor`; the Home Manager copy in
`modules/desktop/home/options.nix` already mirrors that same path via
`helpers.mkHomeOpt` reading `osConfig` (`lib/builders.nix`). The value flows
profile → NixOS → HM automatically. Standalone HM (`mkHome`, currently unused)
falls back to the literal default.

### Domain A — power and thermal

New module `modules/general/nixos/power.nix`; options under `nixos.general.power`
in `modules/general/options.nix`.

| option | laptop | desktop | wsl |
|---|---|---|---|
| `governor` | `powersave` | `performance` | — (module off) |
| `autoCpufreq.enable` | true | false | false |
| `powertop.enable` | true | false | false |
| `thermald.enable` | true | true | false |
| `upower.enable` | true | false | false |

Two deliberate behaviour changes:

- **One owner for the governor.** The desktop runs the static `performance`
  governor with auto-cpufreq off; the laptop runs auto-cpufreq and does not set
  a static governor. Today both hosts set both.
- **powertop off on the desktop.** Its tunings are battery-oriented and its USB
  autosuspend rules are a common source of flaky peripherals on a workstation.

`thermald` is Intel-only; the option documents that rather than trying to detect
the CPU vendor at eval time.

### Domain B — networking posture

`profiles/network.nix` folds into the existing `modules/general/nixos/network.nix`
(which already carries the wpa_supplicant OpenSSL workaround).
`profiles/firewall.nix` becomes a new `modules/general/nixos/firewall.nix`. Both
profile fragments are deleted and the laptop's imports updated.

Options under `nixos.general.network` and `nixos.general.firewall`:

| option | laptop | desktop | wsl |
|---|---|---|---|
| `network.manager.enable` | true | true | false |
| `network.wifi.enable` | true | false | false |
| `network.firmware.enable` | true | true | false |
| `firewall.enable` | true | true | false |
| `firewall.allowedTCPPorts` | `[ 22 80 443 ]` | `[ 22 80 443 ]` | — |
| `firewall.trustedSubnets` | `[ ]` | `[ "192.168.0.0/24" ]` | — |

Concrete effects, so the options are not open to interpretation:

- `network.manager.enable` → `networking.networkmanager.enable` plus the
  openvpn/openconnect plugins.
- `network.wifi.enable` → whether this host has wireless hardware for
  NetworkManager to manage. Both hosts keep `networking.wireless.enable = false`
  (standalone wpa_supplicant conflicts with NM, which starts its own); the flag
  gates NM's wifi backend settings and is the single place a wifi-less host is
  declared.
- `network.firmware.enable` → `hardware.enableRedistributableFirmware`. Stays
  true on the desktop: it is needed for GPU and microcode, not only wifi.

`trustedSubnets` replaces the dead hardcoded `192.168.1.0/24` rule and is
rendered through the firewall backend actually in use.

The desktop currently opens no ports; adopting the shared default means it will
accept 22/80/443. This is intended (confirmed 2026-07-25).

Wake-on-LAN is deliberately **not** added — no current need, and an unused
option is a liability.

### Domain C — Eww bar

- `home.desktop.hyprland.eww.laptopControls.enable` defaults to
  `config.formFactor == "laptop"`; both profiles stop setting it by hand. The
  `eww-lid-inhibit` user service (`eww/default.nix:163`) already keys off it, so
  lid handling follows automatically.
- New `home.desktop.hyprland.eww.battery.enable`, default
  `config.formFactor == "laptop"`. Threaded into `eww.yuck` as a
  `@batteryModule@` substitution alongside the existing `@laptopControls@`
  (`pkgs.replaceVars` in `eww/default.nix`), gating the bar button and the
  battery popup, and letting the backend skip the battery collector entirely.
- **`network_connection_state()` gains a non-NM fallback.** When `nmcli`
  produces nothing usable, read link state and the default route directly and
  report the real interface with a distinct `degraded` class. `⚠ Disconnected`
  becomes reserved for genuinely no carrier and no default route.

  Widened during implementation, from review: the trigger is not "nmcli
  returned nothing" but "nmcli reports nothing connected". NetworkManager can
  be running while owning nothing (unmanaged devices, systemd-networkd, or a
  tunnel holding the route), which produced the same false "Disconnected". The
  fallback therefore sits at the terminal branch rather than behind an
  empty-output check.

  The tooltip reads `(kernel routing table; NetworkManager not reporting)`
  rather than the originally specified `(NetworkManager not running)`:
  `run_text()` flattens "not running", "not on PATH" and "timed out" into the
  same empty string, so naming one cause would be asserting what it cannot
  know.

- **`eww.wifi.enable`** (added during implementation) hides the Wi-Fi toggle in
  the network popup on a host with no wireless, and `network_popup`'s fixed
  height is templated alongside it (200px with the row, 160px without) so the
  shorter panel does not leave a dead band inside the card.
- **Interface selection becomes route-first**: prefer the device that owns the
  default route, then fall back to the existing wifi-then-ethernet order. This
  is correct on both hosts and needs no new option.
- `eww.scss` gains styling for the `degraded` class.

### Domain D — idle / lid / sleep

`services.hypridle` moves behind options under `home.desktop.hyprland.idle`:

| option | laptop | desktop |
|---|---|---|
| `lockTimeout` | 900 | 900 |
| `dpmsTimeout` | 1200 | 1200 |
| `suspend.enable` | true | **false** |
| `suspendTimeout` | 1800 | (unused) |

With `suspend.enable = false` the third listener is omitted entirely, so the
desktop blanks its screens but never suspends.

## Testing Strategy

Every commit is test-driven: the test lands first and fails for the right
reason, then the implementation makes it pass.

### Nix side — eval assertions

New `checks.<system>.host-options` in `flake.nix`: a derivation that evaluates
all three `nixosConfigurations` and asserts derived option values, failing with
a readable diff of every mismatch. This is the red/green mechanism for Nix
changes — before the option exists the check fails to evaluate; after
implementation it evaluates and passes.

Assertions cover, per host: the power table, the network/firewall table, the
Eww option defaults, and the hypridle listener set (specifically that the
desktop's configuration contains no suspend listener).

### Python side — unittest

The existing suite (`tests/eww_bar_backend/`, `unittest` + `unittest.mock`)
gains cases in `test_network.py` for the NM-down fallback and route-first
selection, and a case asserting the battery collector is skipped when the module
is disabled.

These tests are not currently reachable from `nix flake check` or CI, so a new
`checks.<system>.eww-backend` derivation runs `python -m unittest discover -s tests`.
Both checks then run in CI via the existing `nix flake check` step.

A `just check` recipe (`nix flake check`) is added so both suites are runnable
in one command locally.

## Implementation Stages

Five commits, each `just fmt` → `just check` → `just test` → `just dry-build`:

1. `feat(options): add formFactor and derive power capabilities` — assertions
   first, then `modules/options.nix`, `modules/general/options.nix`, new
   `modules/general/nixos/power.nix`; profiles set `formFactor` and drop their
   inline power blocks. Includes the `just check` recipe and the
   `host-options` check scaffold.
2. `refactor(network): own network and firewall policy in modules` — assertions
   first, then absorb both profile fragments, add `trustedSubnets`, delete
   `profiles/network.nix` and `profiles/firewall.nix`, update the laptop's
   imports.
3. `fix(eww): distinguish a dead NetworkManager from being offline` — Python
   tests first, then the fallback and route-first selection, plus the
   `degraded` style and the `eww-backend` check.
4. `feat(eww): derive laptopControls and battery module from formFactor` —
   tests first, then options, `@batteryModule@` gating, collector skip.
5. `feat(hypr): form-factor-aware idle policy` — assertions first, then
   hypridle behind options.

Commit 3 stands alone deliberately: it fixes an observed bug, applies to both
hosts, and should not wait behind the refactor.

`just build` is run by the user, not by the implementer.

## Verification

- `just fmt` — nixfmt is enforced by the `pre-commit-check` hook in `flake.nix:119`.
- `just check` — `nix flake check`: pre-commit, `host-options`, `eww-backend`.
- `just test` — eval of the current host's toplevel.
- `just dry-build` — closure diff for the current host.
- CI (`.github/workflows/ci.yml`) additionally dry-builds wsl, laptop and
  desktop, so every commit must keep all three evaluating.

## Docs to Update

- `profiles/README.md` — the `formFactor` convention and what adding a host now
  requires.
- `modules/README.md` and `modules/general/README.md` — the new power and
  firewall modules.
- `.claude/skills/nixos-config/SKILL.md` — the invariant (modules read
  capability options; only `options.nix` files read `formFactor`) belongs in
  "Modular Philosophy".

## Out of Scope

- Samba (`profiles/smb.nix`) stays laptop-only.
- Wake-on-LAN, sshd, and other new desktop capabilities.
- NVIDIA/CUDA settings in `profiles/desktop/nvidia.nix`.
- Repairing the currently stopped units from the 2026-07-25 switch — that is an
  operational fix (reboot), not a config change.

## Risks

- Commits 2 and 5 change live networking and session behaviour. The machine is
  currently in a half-switched state with ten units stopped; reboot before
  landing commit 2 so the switch starts from a consistent baseline.
- Landing commit 2 restarts NetworkManager: expect a brief network drop and a
  bar flicker.
- The `nft`-in-`extraCommands` question means the firewall rule needs live
  confirmation (`systemctl status firewall`) after commit 2, not just a passing
  eval.
