{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.general-settings.secrets.enable {
    # Nothing for now
  };
}
