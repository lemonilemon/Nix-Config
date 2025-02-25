{
  config,
  lib,
  ...
}:
{
  imports = [
    ./desktop
    ./browsers
    ./kitty.nix
    ./apps.nix
  ];
  options = {
    gui-settings.enable = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable my GUI settings";
    };
  };
  config = lib.mkIf config.gui-settings.enable { };
}
