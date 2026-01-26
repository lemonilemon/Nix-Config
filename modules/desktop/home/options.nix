{
  lib,
  config,
  osConfig ? null,
  helpers,
  ...
}:
{
  options = {
    home.desktop = {
      enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.enable";
        default = config.home.enable && config.desktop.enable;
        description = "Enable desktop environment configuration for home-manager modules";
      };

      hyprland.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.enable";
        default = config.home.desktop.enable;
        description = "Enable hyprland for desktop environment";
      };
    };
  };
}
