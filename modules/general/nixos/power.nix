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
      # mkDefault so a desktop environment that wants these can win: the GNOME
      # module sets services.upower.enable at normal priority, which would
      # otherwise collide with ours and fail to evaluate.
      thermald.enable = lib.mkDefault cfg.thermald.enable;
      upower.enable = lib.mkDefault cfg.upower.enable;
      # Normal priority on purpose, unlike the two above: desktop environments
      # set this to mkDefault true, and two mkDefaults would conflict. A plain
      # false beats theirs, which is what we want -- it fights both auto-cpufreq
      # and the static governor.
      power-profiles-daemon.enable = false;
      auto-cpufreq = {
        inherit (cfg.autoCpufreq) enable settings;
      };
    };
  };
}
