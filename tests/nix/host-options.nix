{ pkgs, nixosConfigurations }:
# Assertions live here only when the failure they catch would be SILENT --
# something that still evaluates, still builds, still boots, and is simply
# wrong. Anything a wrong value would announce by itself (an eval error, a
# build failure, a missing widget, a machine that suspends when it should not)
# is left to announce itself.
#
# This file used to also restate derived option values back at the config:
# `powerManagement.cpuFreqGovernor == "performance"`, port lists, per-host
# eww toggles. Those can only fail when you deliberately change the config,
# at which point you update the expectation to match -- so they reported
# nothing the diff had not already shown, and cost an edit on every change.
# They were removed rather than maintained. Do not add more of that shape:
# an expectation copied out of `nix eval` output is testing itself.
let
  inherit (pkgs) lib;

  hosts = {
    desktop = nixosConfigurations.desktop.config;
    laptop = nixosConfigurations.laptop.config;
    wsl = nixosConfigurations.NixOS-wsl.config;
  };

  # A desktop with the programs flag off. Proves home.cli.programs.enable
  # actually gates modules/cli/home/programs/, rather than only gating
  # default.nix's own config block.
  programsOff =
    (nixosConfigurations.desktop.extendModules {
      modules = [ { home.cli.programs.enable = false; } ];
    }).config.home-manager.users.lemonilemon;

  programsOffPackageNames = map (p: p.pname or p.name or "") programsOff.home.packages;

  desktopHome = hosts.desktop.home-manager.users.lemonilemon;

  programsOnPackageNames = map (p: p.pname or p.name or "") desktopHome.home.packages;

  # Every agent instruction target must resolve to one store object, so the
  # tools cannot drift apart. Paths were confirmed against the installed
  # binaries; the kiro one is inferred and needs rechecking after its first run.
  agentInstructionSources = [
    desktopHome.home.file."AGENTS.md".source
    desktopHome.home.file.".claude/CLAUDE.md".source
    desktopHome.home.file.".codex/AGENTS.md".source
    desktopHome.home.file.".kiro/steering/00-global.md".source
    desktopHome.xdg.configFile."opencode/AGENTS.md".source
  ];

  expectations = [
    # --- firewall: rules that fail open ---
    {
      # Witnesses that trustedSubnets reaches the input chain at all. If the
      # concatMapStringsSep in modules/general/nixos/firewall.nix broke, the
      # subnet would quietly stop being trusted with nothing to show for it.
      name = "desktop/firewall rule mentions the LAN";
      actual = lib.hasInfix "192.168.0.0/24" hosts.desktop.networking.firewall.extraCommands;
      expected = true;
    }
    {
      # Behaviour change, not just a longer list: profiles/firewall.nix wrote
      # this as lib.mkDefault, and types.listOf filters overrides before
      # concatenating, so Samba's normal-priority [139 445] discarded 22/80/443
      # outright. The laptop never actually had them open; now it does.
      name = "laptop/firewall.allowedTCPPorts (option)";
      actual = hosts.laptop.nixos.general.firewall.allowedTCPPorts;
      expected = [
        22
        80
        443
      ];
    }
    {
      # The rule this replaced named a subnet neither machine is on, and went
      # through nft under an iptables backend. Guard against it coming back.
      name = "laptop/firewall has no stale 192.168.1.0/24 rule";
      actual = lib.hasInfix "192.168.1.0/24" hosts.laptop.networking.firewall.extraCommands;
      expected = false;
    }
    {
      # The capability option is off for wsl, so our module contributes nothing
      # and NixOS' own default stands. Written down so the profiles/README table
      # is not read as "wsl has no firewall".
      name = "wsl/networking.firewall.enable (NixOS default, module inert)";
      actual = hosts.wsl.networking.firewall.enable;
      expected = true;
    }

    # --- idle policy ---
    {
      # A property of the whole listener list rather than a value: no suspend
      # action may exist on a desktop, however the listeners are rearranged.
      name = "desktop/hypridle has no suspend action";
      actual = builtins.any (
        l: (l.on-timeout or "") == "systemctl suspend"
      ) hosts.desktop.home-manager.users.lemonilemon.services.hypridle.settings.listener;
      expected = false;
    }

    # --- home.cli.programs.enable actually gates programs/ ---
    # Positive controls: prove claude-code and bubblewrap actually exist in
    # home.packages when the flag is on, so the absence checks below can't go
    # vacuous (e.g. an upstream rename of llm-agents' claude-code pname) and
    # silently stop testing anything.
    {
      name = "desktop/claude-code in home.packages";
      actual = builtins.any (n: n == "claude-code") programsOnPackageNames;
      expected = true;
    }
    {
      name = "desktop/bubblewrap in home.packages";
      actual = builtins.any (n: n == "bubblewrap") programsOnPackageNames;
      expected = true;
    }
    {
      name = "desktop/zip in home.packages";
      actual = builtins.any (n: n == "zip") programsOnPackageNames;
      expected = true;
    }
    {
      name = "programs off/atuin.enable";
      actual = programsOff.programs.atuin.enable;
      expected = false;
    }
    {
      name = "programs off/fastfetch.enable";
      actual = programsOff.programs.fastfetch.enable;
      expected = false;
    }
    {
      name = "programs off/opencode.enable";
      actual = programsOff.programs.opencode.enable;
      expected = false;
    }
    {
      name = "programs off/yazi.enable";
      actual = programsOff.programs.yazi.enable;
      expected = false;
    }
    {
      name = "programs off/claude-code in home.packages";
      actual = builtins.any (n: n == "claude-code") programsOffPackageNames;
      expected = false;
    }
    {
      # bubblewrap, not socat: the eww module puts socat in home.packages of
      # its own accord (eww/default.nix:127), so it survives the programs flag
      # being off and cannot witness anything about ai.nix.
      name = "programs off/bubblewrap in home.packages";
      actual = builtins.any (n: n == "bubblewrap") programsOffPackageNames;
      expected = false;
    }
    {
      # utils.nix only sets home.packages (no enable option to witness
      # directly). zip is a genuine stand-in: it is referenced as a package
      # nowhere else in this repo.
      name = "programs off/zip in home.packages (utils.nix)";
      actual = builtins.any (n: n == "zip") programsOffPackageNames;
      expected = false;
    }

    # --- global agent instructions ---
    {
      name = "agent instructions/all five targets share one store path";
      actual = builtins.all (
        s: builtins.toString s == builtins.toString (builtins.head agentInstructionSources)
      ) agentInstructionSources;
      expected = true;
    }
    {
      name = "agent instructions/source points at the instruction file";
      actual = lib.hasInfix "lemonilemon's agent instructions" (
        builtins.readFile (builtins.head agentInstructionSources)
      );
      expected = true;
    }
    {
      # The flag gates the instruction files too, not just the packages.
      name = "programs off/no agent instruction files";
      actual = programsOff.home.file ? "AGENTS.md";
      expected = false;
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
