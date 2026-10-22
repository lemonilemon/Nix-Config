{
  lib,
  config,
  pkgs,
  ...
}:
let
  cfg = config.home.desktop.hyprland.eww;
in
{
  imports = [ ../../../../../packages/eww-bar/module.nix ];

  config = lib.mkIf cfg.enable {
    programs.eww-bar = {
      enable = true;
      colors = import ../theme/palette.nix;
      laptopControls.enable = cfg.laptopControls.enable;
      battery.enable = cfg.battery.enable;
      wifi.enable = cfg.wifi.enable;
      inherit (cfg) modules;
      defaultHidden = cfg.defaultHiddenModules;
      ai.ccusage.package = pkgs.llm-agents.ccusage;
      ai.openusage.package = pkgs.openusage-cli;
    };
  };
}
