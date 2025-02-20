{
  config,
  lib,
  ...
}:
{
  imports = [
    ./firefox.nix
  ];
  options = {
    gui-settings.browsers.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.gui-settings.enable;
      description = "Enable the browsers module";
    };
  };
  config = lib.mkIf config.gui-settings.browsers.enable {
    gui-settings.browsers.firefox.enable = true;
  };

}
