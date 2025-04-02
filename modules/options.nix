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
  };
}
