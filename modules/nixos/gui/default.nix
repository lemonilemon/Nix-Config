{
  config,
  lib,
  ...
}:
{
  imports = [
    ./desktop
    ./apps.nix
  ];
  options = {
    gui-settings.enable = lib.mkOption {
      type = lib.types.bool;
      default = lib.mkDefault false;
      description = "Enable my GUI settings";
    };
  };
  config = lib.mkIf config.gui-settings.enable {
    # All of the gui settings are default to be enabled
    # gui-settings = {
    # };
  };

}
