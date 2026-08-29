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

  # The two independently-derived views of where the vault is: the editor's and
  # the sync daemon's.
  nvimVaultPaths = map (w: w.path) desktopHome.programs.nixvim.plugins.obsidian.settings.workspaces;
  syncthingVaultPath = hosts.desktop.services.syncthing.settings.folders."obsidian-notes".path;

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
      # Every other capability under nixos.general defaults to following
      # nixos.general.enable. smb deliberately does not, and the reason is
      # security rather than taste: its media share is served to guests with
      # no authentication, and samba's openFirewall binds 139/445 and 137/138
      # on every interface rather than the local subnet. An edit that made it
      # match its siblings would republish ~/Media to strangers with nothing
      # anywhere reporting it.
      name = "laptop/smb.enable (opt-in, off by default)";
      actual = hosts.laptop.nixos.general.smb.enable;
      expected = false;
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

    # --- syncthing: identity collisions are invisible ---
    {
      # Both physical hosts decrypt their identity out of one sops file, keyed
      # by nixos.general.syncthing.deviceName. If that value ever collided --
      # a profile setting it by hand, or the formFactor default changing --
      # both machines would install the same cert, present the same device ID,
      # and each would see the other as itself. Syncthing reports no error for
      # that: it just never syncs, forever, with a healthy-looking UI.
      name = "syncthing/desktop and laptop have distinct identities";
      actual =
        hosts.desktop.sops.secrets."syncthing-cert".key != hosts.laptop.sops.secrets."syncthing-cert".key;
      expected = true;
    }
    {
      # The peer list is the mesh minus self. A filter that stopped removing
      # self would still evaluate and still build, and the resulting config is
      # accepted by Syncthing -- it would simply carry a dead peer entry it
      # can never connect to.
      name = "syncthing/no host lists itself as a peer";
      actual =
        builtins.any (h: h.services.syncthing.settings.devices ? "${h.nixos.general.syncthing.deviceName}")
          [
            hosts.desktop
            hosts.laptop
          ];
      expected = false;
    }

    {
      # This one has already gone wrong once: obsidian.nvim pointed at
      # ~/obsidian/school, a directory that did not exist, while the real vault
      # lived elsewhere. Nothing reported it -- the plugin simply operated on a
      # path nothing else knew about. Both values now derive from
      # home.general.obsidian.vaultPath, and this witnesses that they still do,
      # so a future edit cannot re-separate the editor from the synced folder.
      name = "obsidian/nvim workspace and syncthing folder are the same path";
      actual = nvimVaultPaths;
      expected = [ syncthingVaultPath ];
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
