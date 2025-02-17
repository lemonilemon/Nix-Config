{
  lib,
  pkgs,
  config,
  ...
}:
{
  options = {
    general-settings.utils = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.general-settings.enable;
        description = "Enable my settings of utilities";
      };
    };
  };
  config = {
    home.packages = with pkgs; [
      just
    ];
  };
}
