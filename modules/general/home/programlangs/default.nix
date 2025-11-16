{
  lib,
  config,
  ...
}:
{

  config = lib.mkIf config.home.general.programlangs.enable {
    home.packages = config.home.general.programlangs.packages;
  };
}
