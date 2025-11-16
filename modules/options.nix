{
  lib,
  ...
}:
{
  options = {
    home.enable = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable my home-manager settings";
    };
    nixos.enable = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable my NixOS settings";
    };
    cli.enable = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable my CLI settings";
    };
    gui.enable = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable my GUI settings";
    };
    general.enable = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable my general settings";
    };
    desktop.enable = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable my desktop environment settings";
    };
  };
}
