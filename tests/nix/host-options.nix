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
