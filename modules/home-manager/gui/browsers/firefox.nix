{
  config,
  lib,
  ...
}:
{
  options = {
    gui-settings.browsers.firefox.enable = lib.mkOption {
      type = lib.types.bool;
      default = lib.mkDefault config.gui-settings.browsers.enable;
      description = "Enable firefox for browsing";
    };
  };
  config = lib.mkIf config.gui-settings.browsers.firefox.enable {
    programs.firefox.enable = true;
  };
}
