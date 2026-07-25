{ pkgs, nixosConfigurations }:
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

  programsOnPackageNames = map (p: p.pname or p.name or "") (
    hosts.desktop.home-manager.users.lemonilemon.home.packages
  );

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
    {
      name = "laptop/services.auto-cpufreq.enable";
      actual = hosts.laptop.services.auto-cpufreq.enable;
      expected = true;
    }
    {
      name = "laptop/services.auto-cpufreq.settings";
      actual = hosts.laptop.services.auto-cpufreq.settings;
      expected = {
        battery = {
          governor = "powersave";
          turbo = "never";
        };
        charger = {
          governor = "powersave";
          turbo = "auto";
        };
      };
    }
    {
      name = "laptop/services.upower.enable";
      actual = hosts.laptop.services.upower.enable;
      expected = true;
    }
    {
      name = "desktop/services.thermald.enable";
      actual = hosts.desktop.services.thermald.enable;
      expected = true;
    }
    {
      name = "desktop/powerManagement.powertop.enable";
      actual = hosts.desktop.powerManagement.powertop.enable;
      expected = false;
    }
    {
      name = "wsl/powerManagement.enable";
      actual = hosts.wsl.powerManagement.enable;
      expected = false;
    }

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
      name = "wsl/networking.networkmanager.enable";
      actual = hosts.wsl.networking.networkmanager.enable;
      expected = false;
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
      name = "laptop/firewall.allowedTCPPorts (effective, incl. samba)";
      actual = hosts.laptop.networking.firewall.allowedTCPPorts;
      expected = [
        22
        80
        139
        443
        445
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
    {
      # The desktop must not carry a suspend action at all.
      name = "desktop/hypridle has no suspend action";
      actual = builtins.any (
        l: (l.on-timeout or "") == "systemctl suspend"
      ) hosts.desktop.home-manager.users.lemonilemon.services.hypridle.settings.listener;
      expected = false;
    }
    {
      name = "laptop/hypridle suspends after 1800s";
      actual = builtins.any (
        l: (l.on-timeout or "") == "systemctl suspend" && l.timeout == 1800
      ) hosts.laptop.home-manager.users.lemonilemon.services.hypridle.settings.listener;
      expected = true;
    }

    # --- derived bar options ---
    {
      name = "laptop/eww.laptopControls.enable";
      actual =
        hosts.laptop.home-manager.users.lemonilemon.home.desktop.hyprland.eww.laptopControls.enable;
      expected = true;
    }
    {
      name = "desktop/eww.laptopControls.enable";
      actual =
        hosts.desktop.home-manager.users.lemonilemon.home.desktop.hyprland.eww.laptopControls.enable;
      expected = false;
    }
    {
      name = "laptop/eww.battery.enable";
      actual = hosts.laptop.home-manager.users.lemonilemon.home.desktop.hyprland.eww.battery.enable;
      expected = true;
    }
    {
      name = "desktop/eww.battery.enable";
      actual = hosts.desktop.home-manager.users.lemonilemon.home.desktop.hyprland.eww.battery.enable;
      expected = false;
    }
    {
      name = "desktop/eww.wifi.enable";
      actual = hosts.desktop.home-manager.users.lemonilemon.home.desktop.hyprland.eww.wifi.enable;
      expected = false;
    }
    {
      name = "laptop/eww.wifi.enable";
      actual = hosts.laptop.home-manager.users.lemonilemon.home.desktop.hyprland.eww.wifi.enable;
      expected = true;
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
