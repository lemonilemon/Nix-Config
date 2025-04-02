{
  lib,
  config,
  ...
}:
{

  config = lib.mkIf config.home.general-settings.programlangs.enable {
    home.packages = config.home.general-settings.programlangs.packages;
  };
}
