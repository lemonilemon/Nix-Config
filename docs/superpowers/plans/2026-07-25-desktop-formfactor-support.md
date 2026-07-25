# Desktop Form-Factor Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `formFactor` ("laptop" | "desktop" | "wsl") the single declared fact that power, networking, Eww bar and idle policy derive from, and fix the bar reporting `⚠ Disconnected` whenever NetworkManager is down.

**Architecture:** One top-level `formFactor` option in `modules/options.nix`. Each domain declares named capability options whose *defaults* are expressions over `config.formFactor`, and a module implements those options. Module `config` blocks never read `formFactor` — only `options.nix` files do. Home Manager values ride the existing `helpers.mkHomeOpt` + `osConfig` mirroring.

**Tech Stack:** Nix flakes, NixOS modules, Home Manager, Eww (yuck + SCSS), Python 3 (`unittest`), just.

**Spec:** `docs/superpowers/specs/2026-07-25-desktop-formfactor-support-design.md`

---

## Background an engineer needs before starting

- `modules/options.nix` is imported by **both** `modules/nixos.nix` and `modules/home.nix`, so an option declared there exists in both trees.
- `helpers.mkHomeOpt` (`lib/builders.nix:5-28`) declares a Home Manager option whose default is read from `osConfig` at the given dot-path, falling back to a literal when running standalone. So a Home Manager option mirrors its NixOS twin automatically as long as the `path` string matches.
- `just test` evaluates the current host only. `NIXHOST` selects that host; it is `desktop` on this machine.
- `just build` switches the live system. **Never run it** — the user runs it.
- The Python backend lives at `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/` and is exercised by `tests/eww_bar_backend/`. Tests compute the repo root as `parents[2]` of the test file, so the directory layout matters.
- `eww.yuck` is templated through `pkgs.replaceVars` (`eww/default.nix:72-74`). Every `@name@` placeholder in the file must appear in that attribute set.

---

## Task 1: `formFactor` and derived power capabilities

**Files:**
- Create: `tests/nix/host-options.nix`
- Create: `modules/general/nixos/power.nix`
- Modify: `flake.nix:118-128` (checks)
- Modify: `Justfile` (add `check` recipe)
- Modify: `modules/options.nix:36`
- Modify: `modules/general/options.nix` (inside `nixos.general`)
- Modify: `modules/general/nixos/default.nix`
- Modify: `profiles/desktop/default.nix:31-64`
- Modify: `profiles/laptop/default.nix:25-63`

- [ ] **Step 1: Write the failing assertion check**

Create `tests/nix/host-options.nix`:

```nix
{ pkgs, nixosConfigurations }:
let
  inherit (pkgs) lib;

  hosts = {
    desktop = nixosConfigurations.desktop.config;
    laptop = nixosConfigurations.laptop.config;
    wsl = nixosConfigurations.NixOS-wsl.config;
  };

  expectations = [
    # --- form factor ---
    {
      name = "desktop/formFactor";
      actual = hosts.desktop.formFactor;
      expected = "desktop";
    }
    {
      name = "laptop/formFactor";
      actual = hosts.laptop.formFactor;
      expected = "laptop";
    }
    {
      name = "wsl/formFactor";
      actual = hosts.wsl.formFactor;
      expected = "wsl";
    }

    # --- derived power options ---
    {
      name = "desktop/power.governor";
      actual = hosts.desktop.nixos.general.power.governor;
      expected = "performance";
    }
    {
      name = "laptop/power.governor";
      actual = hosts.laptop.nixos.general.power.governor;
      expected = "powersave";
    }
    {
      name = "desktop/power.autoCpufreq.enable";
      actual = hosts.desktop.nixos.general.power.autoCpufreq.enable;
      expected = false;
    }
    {
      name = "laptop/power.autoCpufreq.enable";
      actual = hosts.laptop.nixos.general.power.autoCpufreq.enable;
      expected = true;
    }
    {
      name = "desktop/power.powertop.enable";
      actual = hosts.desktop.nixos.general.power.powertop.enable;
      expected = false;
    }
    {
      name = "laptop/power.powertop.enable";
      actual = hosts.laptop.nixos.general.power.powertop.enable;
      expected = true;
    }
    {
      name = "desktop/power.upower.enable";
      actual = hosts.desktop.nixos.general.power.upower.enable;
      expected = false;
    }
    {
      name = "laptop/power.upower.enable";
      actual = hosts.laptop.nixos.general.power.upower.enable;
      expected = true;
    }
    {
      name = "wsl/power.enable";
      actual = hosts.wsl.nixos.general.power.enable;
      expected = false;
    }

    # --- effective NixOS config ---
    {
      name = "desktop/services.upower.enable";
      actual = hosts.desktop.services.upower.enable;
      expected = false;
    }
    {
      name = "desktop/services.auto-cpufreq.enable";
      actual = hosts.desktop.services.auto-cpufreq.enable;
      expected = false;
    }
    {
      name = "desktop/powerManagement.cpuFreqGovernor";
      actual = hosts.desktop.powerManagement.cpuFreqGovernor;
      expected = "performance";
    }
    {
      # auto-cpufreq owns the governor on the laptop, so nothing sets it statically.
      name = "laptop/powerManagement.cpuFreqGovernor";
      actual = hosts.laptop.powerManagement.cpuFreqGovernor;
      expected = null;
    }
    {
      name = "laptop/powerManagement.powertop.enable";
      actual = hosts.laptop.powerManagement.powertop.enable;
      expected = true;
    }
  ];

  failures = builtins.filter (e: e.actual != e.expected) expectations;

  report = lib.concatMapStringsSep "\n" (
    e: "  ${e.name}: expected ${builtins.toJSON e.expected}, got ${builtins.toJSON e.actual}"
  ) failures;
in
if failures == [ ] then
  pkgs.runCommand "host-options-assertions" { } "touch $out"
else
  throw "host-options assertions failed:\n${report}"
```

- [ ] **Step 2: Wire the check into the flake**

In `flake.nix`, replace the `checks` block (currently lines 118-128) with:

```nix
      checks = eachSystem (system: {
        pre-commit-check = inputs.pre-commit-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            nixfmt = {
              enable = true;
              package = nixpkgs.legacyPackages.${system}.nixfmt;
            };
          };
        };

        host-options = import ./tests/nix/host-options.nix {
          pkgs = nixpkgs.legacyPackages.${system};
          inherit (self) nixosConfigurations;
        };
      });
```

- [ ] **Step 3: Add the `just check` recipe**

In `Justfile`, directly above the `# Run eval tests` comment for the `test` recipe:

```just
# Run flake checks (nixfmt, host option assertions, backend tests)
[group('nix')]
check:
    nix flake check
```

- [ ] **Step 4: Run the check to verify it fails**

Run: `just check`
Expected: FAIL. The error names the missing option, e.g.
`error: attribute 'formFactor' missing` (the assertion file references options that do not exist yet).

- [ ] **Step 5: Declare `formFactor`**

In `modules/options.nix`, add inside the `options` set, after the `desktop.enable` block (line 32-36):

```nix
    formFactor = lib.mkOption {
      type = lib.types.enum [
        "laptop"
        "desktop"
        "wsl"
      ];
      default = "desktop";
      description = "Physical form factor of this host; seeds power, network, idle and bar defaults.";
    };
```

- [ ] **Step 6: Declare the power capability options**

In `modules/general/options.nix`, add inside the `nixos.general` set, after the `nix` block:

```nix
      power = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.enable && config.formFactor != "wsl";
          description = "Enable my power management settings";
        };

        governor = lib.mkOption {
          type = lib.types.str;
          default = if config.formFactor == "laptop" then "powersave" else "performance";
          description = "CPU frequency governor, applied only when auto-cpufreq is not managing it";
        };

        autoCpufreq = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.formFactor == "laptop";
            description = "Enable auto-cpufreq; it owns the governor at runtime when on";
          };

          settings = lib.mkOption {
            type = lib.types.attrs;
            default =
              if config.formFactor == "laptop" then
                {
                  battery = {
                    governor = "powersave";
                    turbo = "never";
                  };
                  charger = {
                    governor = "powersave";
                    turbo = "auto";
                  };
                }
              else
                { };
            description = "Settings passed through to services.auto-cpufreq";
          };
        };

        powertop.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.formFactor == "laptop";
          description = "Enable powertop tunings; battery-oriented, off on desktops";
        };

        thermald.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.formFactor != "wsl";
          description = "Enable thermald; Intel-only, inert on AMD hardware";
        };

        upower.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.formFactor == "laptop";
          description = "Enable UPower battery reporting";
        };
      };
```

- [ ] **Step 7: Implement the power module**

Create `modules/general/nixos/power.nix`:

```nix
{
  lib,
  config,
  ...
}:
let
  cfg = config.nixos.general.power;
in
{
  config = lib.mkIf cfg.enable {
    powerManagement = {
      enable = true;
      powertop.enable = cfg.powertop.enable;
      # auto-cpufreq rewrites the governor at runtime, so exactly one of the two
      # owns it: the static governor applies only when auto-cpufreq is off.
      cpuFreqGovernor = lib.mkIf (!cfg.autoCpufreq.enable) cfg.governor;
    };

    services = {
      thermald.enable = cfg.thermald.enable;
      upower.enable = cfg.upower.enable;
      power-profiles-daemon.enable = false;
      auto-cpufreq = {
        inherit (cfg.autoCpufreq) enable settings;
      };
    };
  };
}
```

- [ ] **Step 8: Import the power module**

In `modules/general/nixos/default.nix`, add `./power.nix` to the imports list:

```nix
{
  imports = [
    ./base.nix
    ./nixld.nix
    ./settings.nix
    ./network.nix
    ./power.nix
  ];
}
```

- [ ] **Step 9: Point the desktop profile at `formFactor`**

In `profiles/desktop/default.nix`, add below the `imports` block:

```nix
  formFactor = "desktop";
```

Then delete the whole `# Power` block (lines 45-50) and the `services = { thermald / power-profiles-daemon / auto-cpufreq }` block (lines 52-64). Do **not** touch `services.blueman.enable = true;` — it is a separate statement further down.

- [ ] **Step 10: Point the laptop profile at `formFactor`**

In `profiles/laptop/default.nix`, add below the `imports` block:

```nix
  formFactor = "laptop";
```

Then delete `services.upower.enable = true;` (line 39), the `# Power` block (lines 40-45) and the `services = { thermald / power-profiles-daemon / auto-cpufreq }` block (lines 47-63). Leave `services.blueman.enable = true;` alone.

- [ ] **Step 11: Run the check to verify it passes**

Run: `just check`
Expected: PASS — no `host-options assertions failed` output.

- [ ] **Step 12: Format and verify the host still evaluates and builds**

Run, in order:

```bash
just fmt
just test
just dry-build
```

Expected: `just test` prints a `.drv` path; `just dry-build` lists the closure diff without switching. The diff should show `powertop` and `auto-cpufreq` leaving the desktop closure.

- [ ] **Step 13: Commit**

```bash
git add flake.nix Justfile tests/nix/host-options.nix modules/options.nix \
  modules/general/options.nix modules/general/nixos/power.nix \
  modules/general/nixos/default.nix profiles/desktop/default.nix profiles/laptop/default.nix
git commit -m "feat(options): add formFactor and derive power capabilities"
```

---

## Task 2: Networking and firewall owned by modules

**Files:**
- Create: `modules/general/nixos/firewall.nix`
- Modify: `tests/nix/host-options.nix` (add expectations)
- Modify: `modules/general/options.nix` (inside `nixos.general`)
- Modify: `modules/general/nixos/network.nix`
- Modify: `modules/general/nixos/default.nix`
- Modify: `profiles/desktop/default.nix:22-29`
- Modify: `profiles/laptop/default.nix:12-23`
- Delete: `profiles/network.nix`, `profiles/firewall.nix`

- [ ] **Step 1: Write the failing assertions**

In `tests/nix/host-options.nix`, append to the `expectations` list (before the closing `];`):

```nix
    # --- derived network options ---
    {
      name = "desktop/network.manager.enable";
      actual = hosts.desktop.nixos.general.network.manager.enable;
      expected = true;
    }
    {
      name = "desktop/network.wifi.enable";
      actual = hosts.desktop.nixos.general.network.wifi.enable;
      expected = false;
    }
    {
      name = "laptop/network.wifi.enable";
      actual = hosts.laptop.nixos.general.network.wifi.enable;
      expected = true;
    }
    {
      name = "wsl/network.enable";
      actual = hosts.wsl.nixos.general.network.enable;
      expected = false;
    }

    # --- effective networking config ---
    {
      name = "desktop/networking.networkmanager.enable";
      actual = hosts.desktop.networking.networkmanager.enable;
      expected = true;
    }
    {
      name = "laptop/networking.networkmanager.enable";
      actual = hosts.laptop.networking.networkmanager.enable;
      expected = true;
    }
    {
      name = "desktop/firewall.enable";
      actual = hosts.desktop.networking.firewall.enable;
      expected = true;
    }
    {
      name = "desktop/firewall.allowedTCPPorts";
      actual = hosts.desktop.networking.firewall.allowedTCPPorts;
      expected = [
        22
        80
        443
      ];
    }
    {
      name = "desktop/firewall.trustedSubnets";
      actual = hosts.desktop.nixos.general.firewall.trustedSubnets;
      expected = [ "192.168.0.0/24" ];
    }
    {
      name = "desktop/firewall rule mentions the LAN";
      actual = lib.hasInfix "192.168.0.0/24" hosts.desktop.networking.firewall.extraCommands;
      expected = true;
    }
    {
      name = "laptop/firewall.trustedSubnets";
      actual = hosts.laptop.nixos.general.firewall.trustedSubnets;
      expected = [ ];
    }
```

- [ ] **Step 2: Run the check to verify it fails**

Run: `just check`
Expected: FAIL with `error: attribute 'network' missing` (or `firewall`) — the options are not declared yet.

- [ ] **Step 3: Declare the network and firewall options**

In `modules/general/options.nix`, add inside the `nixos.general` set, after the `power` block:

```nix
      network = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.enable && config.formFactor != "wsl";
          description = "Enable my networking settings";
        };

        manager.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.network.enable;
          description = "Manage networking with NetworkManager";
        };

        wifi.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.formFactor == "laptop";
          description = "This host has wireless hardware for NetworkManager to manage";
        };

        firmware.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.network.enable;
          description = "Install redistributable firmware (wifi, GPU, microcode)";
        };
      };

      firewall = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.enable && config.formFactor != "wsl";
          description = "Enable my firewall settings";
        };

        allowedTCPPorts = lib.mkOption {
          type = lib.types.listOf lib.types.port;
          default = [
            22
            80
            443
          ];
          description = "TCP ports accepted from anywhere";
        };

        allowedUDPPorts = lib.mkOption {
          type = lib.types.listOf lib.types.port;
          default = [ ];
          description = "UDP ports accepted from anywhere";
        };

        trustedSubnets = lib.mkOption {
          type = lib.types.listOf lib.types.str;
          default = [ ];
          example = [ "192.168.0.0/24" ];
          description = "IPv4 subnets accepted wholesale on the input chain";
        };
      };
```

- [ ] **Step 4: Fold `profiles/network.nix` into the network module**

Replace the entire contents of `modules/general/nixos/network.nix` with:

```nix
{
  pkgs,
  lib,
  config,
  ...
}:
let
  cfg = config.nixos.general.network;
in
{
  config = lib.mkMerge [
    {
      # Network issues with wpa_supplicant
      systemd.services.wpa_supplicant.environment.OPENSSL_CONF = pkgs.writeText "openssl.cnf" ''
        openssl_conf = openssl_init
        [openssl_init]
        ssl_conf = ssl_sect
        [ssl_sect]
        system_default = system_default_sect
        [system_default_sect]
        Options = UnsafeLegacyRenegotiation
        [system_default_sect]
        CipherString = Default:@SECLEVEL=0
      '';
    }

    (lib.mkIf cfg.enable {
      # NetworkManager starts its own supplicant; the standalone service conflicts.
      networking.wireless.enable = lib.mkDefault false;

      networking.networkmanager = lib.mkIf cfg.manager.enable {
        enable = true;
        plugins = with pkgs; [
          networkmanager-openvpn # OpenVPN support
          networkmanager-openconnect # OpenConnect support
        ];
        wifi.powersave = lib.mkDefault cfg.wifi.enable;
      };

      hardware.enableRedistributableFirmware = lib.mkDefault cfg.firmware.enable;
    })
  ];
}
```

- [ ] **Step 5: Implement the firewall module**

Create `modules/general/nixos/firewall.nix`:

```nix
{
  pkgs,
  lib,
  config,
  ...
}:
let
  cfg = config.nixos.general.firewall;
in
{
  config = lib.mkIf cfg.enable {
    networking.firewall = {
      enable = true;
      inherit (cfg) allowedTCPPorts allowedUDPPorts;

      # NixOS' default firewall backend is iptables and `nixos-fw` is its input
      # chain, so the rules go through iptables rather than nft. IPv4 only,
      # which is how these subnets have always been written.
      extraCommands = lib.concatMapStringsSep "\n" (
        subnet: "iptables -A nixos-fw -s ${subnet} -j nixos-fw-accept"
      ) cfg.trustedSubnets;
    };

    environment.systemPackages = with pkgs; [
      nftables
    ];
  };
}
```

- [ ] **Step 6: Import the firewall module**

In `modules/general/nixos/default.nix`:

```nix
{
  imports = [
    ./base.nix
    ./nixld.nix
    ./settings.nix
    ./network.nix
    ./power.nix
    ./firewall.nix
  ];
}
```

- [ ] **Step 7: Strip the inlined NetworkManager block from the desktop profile**

In `profiles/desktop/default.nix`, delete lines 22-29 (the `# networking.wireless.enable = false;` comment and the whole `networking.networkmanager = { ... };` block) and add in their place:

```nix
  nixos.general.firewall.trustedSubnets = [ "192.168.0.0/24" ];
```

- [ ] **Step 8: Drop the fragment imports from the laptop profile**

In `profiles/laptop/default.nix`, remove `../network.nix` and `../firewall.nix` from the `imports` list. Keep `../smb.nix`. The list becomes:

```nix
  imports = [
    # Include the results of the hardware scan.
    ./hardware-configuration.nix
    ./config.nix
    ../base.nix
    ../i18n.nix
    ../boot.nix
    ../sound.nix
    ../smb.nix
  ];
```

Leave `nixos.general.firewall.trustedSubnets` unset on the laptop: its old rule named `192.168.1.0/24` but was issued via `nft` under the iptables backend and never applied, so an empty list preserves the effective behaviour rather than silently opening a subnet.

- [ ] **Step 9: Delete the absorbed fragments**

```bash
git rm profiles/network.nix profiles/firewall.nix
```

- [ ] **Step 10: Run the check to verify it passes**

Run: `just check`
Expected: PASS.

- [ ] **Step 11: Verify all three hosts still evaluate**

```bash
just fmt
just test
just dry-build
nix eval .#nixosConfigurations.laptop.config.system.build.toplevel.drvPath --show-trace
nix eval .#nixosConfigurations.NixOS-wsl.config.system.build.toplevel.drvPath --show-trace
```

Expected: four `.drv` paths and a closure diff. CI dry-builds all three hosts, so laptop and wsl must evaluate here too.

- [ ] **Step 12: Commit**

```bash
git add tests/nix/host-options.nix modules/general/options.nix \
  modules/general/nixos/network.nix modules/general/nixos/firewall.nix \
  modules/general/nixos/default.nix profiles/desktop/default.nix profiles/laptop/default.nix
git commit -m "refactor(network): own network and firewall policy in modules"
```

---

## Task 3: The bar stops calling a dead NetworkManager "Disconnected"

**Files:**
- Modify: `tests/eww_bar_backend/test_network.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py:1036-1085`
- Modify: `modules/desktop/home/hyprland/eww/eww.scss:280-292`
- Modify: `modules/desktop/home/hyprland/eww/default.nix:33-68` (add `iproute2`)
- Modify: `flake.nix` (add the `eww-backend` check)

- [ ] **Step 1: Write the failing tests**

In `tests/eww_bar_backend/test_network.py`, insert these classes above the `if __name__ == "__main__":` block:

```python
ROUTE_TEXT = (
    "default via 192.168.0.1 dev eno1 proto dhcp src 192.168.0.88 metric 100\n"
    "192.168.0.0/24 dev eno1 proto kernel scope link src 192.168.0.88 metric 100\n"
)

ADDR_TEXT = (
    "1: lo    inet 127.0.0.1/8 scope host lo\n"
    "2: eno1    inet 192.168.0.88/24 brd 192.168.0.255 scope global dynamic eno1\n"
)


class LinkFallbackTests(unittest.TestCase):
    def test_finds_default_route_device(self):
        self.assertEqual(collectors.default_route_device(ROUTE_TEXT), "eno1")

    def test_no_default_route(self):
        self.assertEqual(
            collectors.default_route_device("192.168.0.0/24 dev eno1 scope link\n"), ""
        )

    def test_device_ipv4(self):
        self.assertEqual(collectors.device_ipv4(ADDR_TEXT, "eno1"), "192.168.0.88")

    def test_device_ipv4_missing(self):
        self.assertEqual(collectors.device_ipv4(ADDR_TEXT, "wlan0"), "")

    def test_link_state_reports_route_owner(self):
        state = collectors.link_state_from_text(ROUTE_TEXT, ADDR_TEXT)
        self.assertEqual(state["class"], "degraded")
        self.assertIn("eno1", state["text"])
        self.assertIn("192.168.0.88", state["tooltip"])
        self.assertIn("NetworkManager", state["tooltip"])

    def test_link_state_none_without_default_route(self):
        self.assertIsNone(collectors.link_state_from_text("", ADDR_TEXT))


class NetworkManagerDownTests(unittest.TestCase):
    def _fake_run_text(self, responses):
        def fake(command, **_kwargs):
            return responses.get(tuple(command), "")

        return fake

    def test_falls_back_to_link_state_when_nmcli_is_dead(self):
        responses = {
            ("ip", "route"): ROUTE_TEXT,
            ("ip", "-o", "-4", "addr", "show"): ADDR_TEXT,
        }
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text(responses)
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "degraded")
        self.assertIn("eno1", state["text"])

    def test_disconnected_when_nmcli_dead_and_no_route(self):
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text({})
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "disconnected")

    def test_ethernet_wins_when_it_owns_the_default_route(self):
        responses = {
            ("nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status"): (
                "wlan0:wifi:connected\neno1:ethernet:connected\n"
            ),
            ("ip", "route"): ROUTE_TEXT,
            ("nmcli", "-t", "-f", "IP4.ADDRESS", "dev", "show", "eno1"): (
                "IP4.ADDRESS[1]:192.168.0.88/24\n"
            ),
        }
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text(responses)
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "ethernet")
        self.assertIn("eno1", state["text"])
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `python -m unittest discover -s tests -t . -v`
Expected: FAIL with `AttributeError: module 'eww_bar_backend.collectors' has no attribute 'default_route_device'`.

- [ ] **Step 3: Add the link-state helpers**

In `collectors.py`, insert directly above `def network_connection_state():` (currently line 1036):

```python
def default_route_device(route_text):
    for line in route_text.splitlines():
        fields = line.split()
        if not fields or fields[0] != "default" or "dev" not in fields:
            continue
        return fields[fields.index("dev") + 1]
    return ""


def device_ipv4(addr_text, device):
    # `ip -o -4 addr show` lines look like:
    #   2: eno1    inet 192.168.0.88/24 brd ... scope global dynamic eno1
    for line in addr_text.splitlines():
        fields = line.split()
        if len(fields) >= 4 and fields[1] == device and fields[2] == "inet":
            return fields[3].split("/")[0]
    return ""


def link_state_from_text(route_text, addr_text):
    """Readout for when nmcli cannot answer — usually NetworkManager being down.

    The kernel keeps the lease and the default route after NetworkManager
    stops, so the machine is still online; only the usual source of truth is
    gone. Report the interface that owns the route rather than claiming to be
    disconnected.
    """
    device = default_route_device(route_text)
    if not device:
        return None
    ip_info = device_ipv4(addr_text, device)
    return {
        "text": f"󰌗 {device}",
        "tooltip": f"{device}: {ip_info or 'No IP'} (NetworkManager not running)",
        "class": "degraded",
    }


def link_fallback_state():
    return link_state_from_text(
        run_text(["ip", "route"]),
        run_text(["ip", "-o", "-4", "addr", "show"]),
    )
```

- [ ] **Step 4: Use the fallback and prefer the route owner**

In `collectors.py`, replace the head of `network_connection_state()` — the two lines

```python
    status = run_text(["nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status"])
    wifi_device = connected_device(status, "wifi")
```

with:

```python
    status = run_text(["nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status"])
    if not status.strip():
        fallback = link_fallback_state()
        if fallback:
            return fallback
        return {
            "text": "⚠ Disconnected",
            "tooltip": "No connection",
            "class": "disconnected",
        }

    wifi_device = connected_device(status, "wifi")
    ethernet_device = connected_device(status, "ethernet")
    # Whichever interface carries the default route is the one actually in use.
    if ethernet_device and default_route_device(run_text(["ip", "route"])) == ethernet_device:
        wifi_device = ""
```

Then delete the now-duplicated lookup further down — the line

```python
    ethernet_device = connected_device(status, "ethernet")
```

immediately before `if ethernet_device:` (currently line 1065).

- [ ] **Step 5: Run the tests to verify they pass**

Run: `python -m unittest discover -s tests -t . -v`
Expected: PASS, all tests OK.

- [ ] **Step 6: Style the new state**

In `modules/desktop/home/hyprland/eww/eww.scss`, after the `.network.linked` block (line 289-292):

```scss
.network.degraded {
  background: rgba(249, 226, 175, 0.10);
  color: $peach;
}
```

- [ ] **Step 7: Put `ip` on the bar's runtime PATH**

The collector now shells out to `ip`, which is not in `runtimePackages`. In `modules/desktop/home/hyprland/eww/default.nix`, add `iproute2` to the alphabetical list (between `imagemagick` and `jq`):

```nix
      imagemagick
      iproute2
      jq
```

- [ ] **Step 8: Add the backend test check to the flake**

In `flake.nix`, inside the `checks` set, after `host-options`:

```nix
        eww-backend =
          let
            pkgs = nixpkgs.legacyPackages.${system};
          in
          pkgs.runCommand "eww-backend-tests"
            {
              nativeBuildInputs = [ pkgs.python3 ];
              PYTHONDONTWRITEBYTECODE = "1";
            }
            ''
              mkdir -p modules/desktop/home/hyprland/eww
              cp -R ${./tests} tests
              cp -R ${./modules/desktop/home/hyprland/eww/scripts} \
                modules/desktop/home/hyprland/eww/scripts
              python3 -m unittest discover -s tests -t . -v
              touch $out
            '';
```

- [ ] **Step 9: Verify the check runs the suite**

Run: `just check`
Expected: PASS, with the unittest output visible in the build log (`nix log` if it is silent).

- [ ] **Step 10: Format and verify**

```bash
just fmt
just test
just dry-build
```

- [ ] **Step 11: Commit**

```bash
git add tests/eww_bar_backend/test_network.py flake.nix \
  modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py \
  modules/desktop/home/hyprland/eww/eww.scss \
  modules/desktop/home/hyprland/eww/default.nix
git commit -m "fix(eww): distinguish a dead NetworkManager from being offline"
```

---

## Task 4: Bar modules derive from `formFactor`

**Files:**
- Modify: `tests/nix/host-options.nix`
- Modify: `tests/eww_bar_backend/test_battery.py`
- Modify: `modules/desktop/options.nix:25-30`
- Modify: `modules/desktop/home/options.nix` (eww block)
- Modify: `modules/desktop/home/hyprland/eww/default.nix:72-74,133-151`
- Modify: `modules/desktop/home/hyprland/eww/eww.yuck:3,95-97,270-275`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/common.py`
- Modify: `modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py:1210`
- Modify: `profiles/laptop/config.nix`

- [ ] **Step 1: Write the failing Nix assertions**

Append to `expectations` in `tests/nix/host-options.nix`:

```nix
    # --- derived bar options ---
    {
      name = "laptop/eww.laptopControls.enable";
      actual = hosts.laptop.home.desktop.hyprland.eww.laptopControls.enable;
      expected = true;
    }
    {
      name = "desktop/eww.laptopControls.enable";
      actual = hosts.desktop.home.desktop.hyprland.eww.laptopControls.enable;
      expected = false;
    }
    {
      name = "laptop/eww.battery.enable";
      actual = hosts.laptop.home.desktop.hyprland.eww.battery.enable;
      expected = true;
    }
    {
      name = "desktop/eww.battery.enable";
      actual = hosts.desktop.home.desktop.hyprland.eww.battery.enable;
      expected = false;
    }
    {
      name = "desktop/eww.wifi.enable";
      actual = hosts.desktop.home.desktop.hyprland.eww.wifi.enable;
      expected = false;
    }
    {
      name = "laptop/eww.wifi.enable";
      actual = hosts.laptop.home.desktop.hyprland.eww.wifi.enable;
      expected = true;
    }
```

- [ ] **Step 2: Write the failing Python test**

In `tests/eww_bar_backend/test_battery.py`, add above `if __name__ == "__main__":`:

```python
class BatteryModuleDisabledTests(unittest.TestCase):
    def test_disabled_module_skips_collection(self):
        with unittest.mock.patch.dict("os.environ", {"EWW_BAR_BATTERY": "0"}, clear=False):
            state = collectors.battery_state()
        self.assertEqual(state["status"], "Unknown")

    def test_enabled_by_default(self):
        with unittest.mock.patch.dict("os.environ", {}, clear=False):
            os.environ.pop("EWW_BAR_BATTERY", None)
            self.assertTrue(common.module_enabled("BATTERY"))
```

If `test_battery.py` does not already import them, add `import os`, `import unittest.mock` and extend its `from eww_bar_backend import ...` line to include `common`.

- [ ] **Step 3: Run both suites to verify they fail**

Run: `just check`
Expected: FAIL with `error: attribute 'battery' missing`.

Run: `python -m unittest discover -s tests -t . -v`
Expected: FAIL with `AttributeError: module 'eww_bar_backend.common' has no attribute 'module_enabled'`.

- [ ] **Step 4: Derive the NixOS-side bar options**

In `modules/desktop/options.nix`, replace the `hyprland.eww.laptopControls.enable` block (lines 25-30) with:

```nix
      hyprland.eww.laptopControls.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.formFactor == "laptop";
        description = "Enable laptop-only Eww controls for clamshell and headless display modes";
      };

      hyprland.eww.battery.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.formFactor == "laptop";
        description = "Show the Eww battery module and collect battery state";
      };

      hyprland.eww.wifi.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.general.network.wifi.enable;
        description = "Show the Wi-Fi toggle in the Eww network popup";
      };
```

- [ ] **Step 5: Mirror them into Home Manager**

In `modules/desktop/home/options.nix`, after the existing `hyprland.eww.laptopControls.enable` block:

```nix
      hyprland.eww.battery.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.eww.battery.enable";
        default = false;
        description = "Show the Eww battery module and collect battery state";
      };

      hyprland.eww.wifi.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.eww.wifi.enable";
        default = false;
        description = "Show the Wi-Fi toggle in the Eww network popup";
      };
```

- [ ] **Step 6: Thread both flags into the bar**

In `modules/desktop/home/hyprland/eww/default.nix`, replace the `ewwYuck` binding (lines 72-74):

```nix
  ewwYuck = pkgs.replaceVars ./eww.yuck {
    laptopControls = if cfg.laptopControls.enable then "true" else "false";
    batteryModule = if cfg.battery.enable then "true" else "false";
    wifiControls = if cfg.wifi.enable then "true" else "false";
  };
```

Then, in the `eww-bar` service (line 141-148), extend `Environment` so the Python backend sees the battery flag:

```nix
        Service = {
          Environment = [
            "PATH=${runtimePath}"
            "EWW_BAR_BATTERY=${if cfg.battery.enable then "1" else "0"}"
          ];
          ExecStart = "${pkgs.eww}/bin/eww --force-wayland daemon --no-daemonize";
          ExecStartPost = openBar;
          MemoryAccounting = true;
          Restart = "on-failure";
          RestartSec = "1s";
        };
```

- [ ] **Step 7: Gate the widgets in the yuck**

In `modules/desktop/home/hyprland/eww/eww.yuck`, after line 3 (`(defvar laptop_controls "@laptopControls@")`):

```lisp
(defvar battery_module "@batteryModule@")
(defvar wifi_controls "@wifiControls@")
```

Replace the battery button (lines 95-97) with:

```lisp
          (button :class "module-button battery ${bar_state.battery.class}"
            :visible {battery_module == "true"}
            :timeout "5s" :onclick "eww-popup toggle battery_popup ${output}"
            (label :text {bar_state.battery.text}))
```

And in `network_panel`, give the Wi-Fi toggle button (lines 270-275) a `:visible`:

```lisp
      (button :class "popup-action network-wifi ${bar_state.network.wifi_enabled == 'true' ? 'active' : ''}"
        :visible {wifi_controls == "true"}
        :onclick "eww-barctl --quiet network wifi-toggle"
```

Leave the rest of that button's body untouched.

- [ ] **Step 8: Let the backend skip battery collection**

In `common.py`, add after the `run_text` definition:

```python
def module_enabled(name, default="1"):
    """Bar modules the Nix module switched off are signalled through the env."""
    return os.environ.get(f"EWW_BAR_{name}", default) != "0"
```

In `collectors.py`, extend the `from .common import (...)` list with `module_enabled`, then make `battery_state` return early — replace its first line (currently line 1210-1211):

```python
def battery_state(root=Path("/sys/class/power_supply")):
    if not module_enabled("BATTERY"):
        return BATTERY_DEFAULT.copy()
    for battery in root.glob("BAT*"):
```

The 30-second periodic thread that calls it stays as it is: it also drives notifications, and the early return makes its battery half free.

- [ ] **Step 9: Stop setting `laptopControls` by hand**

In `profiles/laptop/config.nix`, delete the line:

```nix
  home.desktop.hyprland.eww.laptopControls.enable = true;
```

It now follows `formFactor = "laptop"`.

- [ ] **Step 10: Run both suites to verify they pass**

```bash
just check
python -m unittest discover -s tests -t . -v
```

Expected: both PASS.

- [ ] **Step 11: Format and verify**

```bash
just fmt
just test
just dry-build
```

- [ ] **Step 12: Commit**

```bash
git add tests/nix/host-options.nix tests/eww_bar_backend/test_battery.py \
  modules/desktop/options.nix modules/desktop/home/options.nix \
  modules/desktop/home/hyprland/eww/default.nix \
  modules/desktop/home/hyprland/eww/eww.yuck \
  modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/common.py \
  modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/collectors.py \
  profiles/laptop/config.nix
git commit -m "feat(eww): derive laptopControls and battery module from formFactor"
```

---

## Task 5: Form-factor-aware idle policy

**Files:**
- Modify: `tests/nix/host-options.nix`
- Modify: `modules/desktop/options.nix` (`home.desktop` set)
- Modify: `modules/desktop/home/options.nix` (`home.desktop` set)
- Modify: `modules/desktop/home/hyprland/hyprlock.nix:110-137`

- [ ] **Step 1: Write the failing assertions**

Append to `expectations` in `tests/nix/host-options.nix`:

```nix
    # --- idle policy ---
    {
      name = "desktop/idle.suspend.enable";
      actual = hosts.desktop.home.desktop.hyprland.idle.suspend.enable;
      expected = false;
    }
    {
      name = "laptop/idle.suspend.enable";
      actual = hosts.laptop.home.desktop.hyprland.idle.suspend.enable;
      expected = true;
    }
    {
      # lock + dpms only; the suspend listener is dropped entirely.
      name = "desktop/hypridle listener count";
      actual = builtins.length (
        hosts.desktop.home-manager.users.lemonilemon.services.hypridle.settings.listener
      );
      expected = 2;
    }
    {
      name = "laptop/hypridle listener count";
      actual = builtins.length (
        hosts.laptop.home-manager.users.lemonilemon.services.hypridle.settings.listener
      );
      expected = 3;
    }
```

- [ ] **Step 2: Run the check to verify it fails**

Run: `just check`
Expected: FAIL with `error: attribute 'idle' missing`.

- [ ] **Step 3: Declare the idle options (NixOS side)**

In `modules/desktop/options.nix`, add inside the `home.desktop` set, after the eww options:

```nix
      hyprland.idle = {
        lockTimeout = lib.mkOption {
          type = lib.types.int;
          default = 900;
          description = "Seconds of idle before the session locks";
        };

        dpmsTimeout = lib.mkOption {
          type = lib.types.int;
          default = 1200;
          description = "Seconds of idle before the displays are switched off";
        };

        suspend = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.formFactor == "laptop";
            description = "Suspend the machine after suspendTimeout of idle";
          };
        };

        suspendTimeout = lib.mkOption {
          type = lib.types.int;
          default = 1800;
          description = "Seconds of idle before suspending, when suspend is enabled";
        };
      };
```

- [ ] **Step 4: Mirror the idle options into Home Manager**

In `modules/desktop/home/options.nix`, add inside the `home.desktop` set:

```nix
      hyprland.idle.lockTimeout = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.idle.lockTimeout";
        type = lib.types.int;
        default = 900;
        description = "Seconds of idle before the session locks";
      };

      hyprland.idle.dpmsTimeout = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.idle.dpmsTimeout";
        type = lib.types.int;
        default = 1200;
        description = "Seconds of idle before the displays are switched off";
      };

      hyprland.idle.suspend.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.idle.suspend.enable";
        default = false;
        description = "Suspend the machine after suspendTimeout of idle";
      };

      hyprland.idle.suspendTimeout = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.idle.suspendTimeout";
        type = lib.types.int;
        default = 1800;
        description = "Seconds of idle before suspending, when suspend is enabled";
      };
```

- [ ] **Step 5: Build the listener list from the options**

In `modules/desktop/home/hyprland/hyprlock.nix`, replace the `services.hypridle` block (lines 110-137) with:

```nix
    services.hypridle =
      let
        idle = config.home.desktop.hyprland.idle;
      in
      {
        enable = true;
        settings = {
          general = {
            lock_cmd = "pidof hyprlock || hyprlock"; # avoid starting multiple hyprlock instances.
            before_sleep_cmd = "loginctl lock-session"; # lock before suspend.
            # Re-enable display then restart hyprlock if it crashed during GPU reset on suspend.
            after_sleep_cmd = "hyprctl dispatch dpms on; pidof hyprlock || hyprlock";
          };
          listener = [
            {
              timeout = idle.lockTimeout;
              on-timeout = "loginctl lock-session";
            }
            {
              # Turn off display while locked to save power.
              # on-resume brings it back if the user wakes before suspend fires.
              timeout = idle.dpmsTimeout;
              on-timeout = "hyprctl dispatch dpms off";
              on-resume = "hyprctl dispatch dpms on";
            }
          ]
          ++ lib.optionals idle.suspend.enable [
            {
              timeout = idle.suspendTimeout;
              on-timeout = "systemctl suspend";
            }
          ];
        };
      };
```

- [ ] **Step 6: Run the check to verify it passes**

Run: `just check`
Expected: PASS.

- [ ] **Step 7: Format and verify**

```bash
just fmt
just test
just dry-build
```

Expected: the closure diff shows a changed `hypridle.conf` for the desktop.

- [ ] **Step 8: Commit**

```bash
git add tests/nix/host-options.nix modules/desktop/options.nix \
  modules/desktop/home/options.nix modules/desktop/home/hyprland/hyprlock.nix
git commit -m "feat(hypr): form-factor-aware idle policy"
```

---

## Task 6: Documentation

**Files:**
- Modify: `profiles/README.md`
- Modify: `modules/README.md`
- Modify: `modules/general/README.md`
- Modify: `.claude/skills/nixos-config/SKILL.md`

- [ ] **Step 1: Document the convention in `profiles/README.md`**

Add this section, placed after the existing description of what a profile contains:

```markdown
## Form Factor

Every profile declares exactly one form factor:

```nix
formFactor = "desktop"; # "laptop" | "desktop" | "wsl"
```

Power, networking, firewall, Eww bar and idle defaults all derive from it — see
`modules/options.nix` for the declaration and the per-domain `options.nix` files
for what each value implies. A host that deviates overrides the single derived
option it disagrees with, for example:

```nix
formFactor = "desktop";
nixos.general.power.governor = "powersave"; # this box runs hot
```

Module `config` blocks never read `formFactor` — they read the capability
options it seeds. Only `options.nix` files mention it.

The former `profiles/network.nix` and `profiles/firewall.nix` fragments are
gone; their content lives in `modules/general/nixos/network.nix` and
`modules/general/nixos/firewall.nix` and is selected by option, not by import.
```

- [ ] **Step 2: Document the new modules**

In `modules/general/README.md`, add to the list of NixOS modules, matching the
surrounding style:

```markdown
- `nixos/power.nix` — CPU governor, auto-cpufreq, powertop, thermald and UPower,
  driven by `nixos.general.power.*`. auto-cpufreq and the static governor are
  mutually exclusive by construction.
- `nixos/firewall.nix` — `networking.firewall` ports and wholesale-trusted IPv4
  subnets, driven by `nixos.general.firewall.*`.
```

In `modules/README.md`, add the same two files to whichever inventory or table
lists the `general` category's modules, with the same one-line descriptions.

- [ ] **Step 3: Record the invariant in the skill**

In `.claude/skills/nixos-config/SKILL.md`, add a third bullet to "Modular Philosophy":

```markdown
3. **Form factor drives defaults**: `formFactor` (`modules/options.nix`) is the one
   declared fact about the machine. Capability options derive their *defaults* from it;
   module `config` blocks read the capability options, never `formFactor` itself. Only
   `options.nix` files mention `formFactor`.
```

- [ ] **Step 4: Commit**

```bash
git add profiles/README.md modules/README.md modules/general/README.md \
  .claude/skills/nixos-config/SKILL.md
git commit -m "docs: document formFactor convention and new modules"
```

---

## Post-implementation verification (user-run)

The user runs these; the implementer does not.

1. Reboot to clear the half-switched state left by the 2026-07-25 rebuild (NetworkManager and nine other units stopped).
2. `just build`
3. Confirm the firewall rule actually loaded: `sudo iptables -S nixos-fw | grep 192.168.0.0/24` and `systemctl status firewall` (must be active, no errors). This is the check the old `nft`-based rule would have failed.
4. Confirm the bar shows the wired connection, and that stopping NetworkManager (`sudo systemctl stop NetworkManager`) now shows the interface with the "NetworkManager not running" tooltip rather than `⚠ Disconnected`. Start it again afterwards.
5. Confirm the desktop bar has no battery module and no Wi-Fi toggle in the network popup.
6. Confirm the desktop no longer suspends: `hyprctl dispatch dpms off` still works, and `~/.config/hypr/hypridle.conf` contains two listeners.
