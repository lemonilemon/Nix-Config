{
  lib,
  config,
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
      defaultHidden = cfg.defaultHiddenModules;
    };
  };
}
