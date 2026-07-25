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
