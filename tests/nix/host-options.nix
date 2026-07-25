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
      # The laptop's own list is the shared default; Samba's openFirewall adds
      # 139/445 on top, which is why the effective list is longer there.
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
