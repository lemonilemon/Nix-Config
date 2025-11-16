{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.general.secrets.enable {
    # Nothing for now
  };
}
