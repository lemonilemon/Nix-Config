{
  lib,
  pkgs,
  config,
  ...
}:
{
  options = {
    general-settings.secrets = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.general-settings.enable;
        description = "Enable my settings of secrets";
      };
    };
  };
  config = lib.mkIf config.general-settings.secrets.enable {
    home.packages = with pkgs; [
      _1password-cli
    ];
  };
}
