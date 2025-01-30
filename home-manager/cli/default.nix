{
  config,
  pkgs,
  lib,
  ...
}:
{
  imports = [ ];
  options = {
    cli-settings.enable = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable my CLI settigns";
    };
  };
  config = lib.mkIf config.cli-settings.enable {

  };

}
