{
  lib,
  config,
  osConfig,
  ...
}:
{
  options = {
    home.desktop = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default =
          if osConfig == null then
            config.home.enable && config.desktop.enable
          else
            osConfig.home.desktop.enable;
        description = "Enable desktop environment configuration for home-manager modules";
      };

      hyprland.enable = lib.mkOption {
        type = lib.types.bool;
        default =
          if osConfig == null then config.home.desktop.enable else osConfig.home.desktop.hyprland.enable;
        description = "Enable hyprland for desktop environment";
      };
    };
  };
}
