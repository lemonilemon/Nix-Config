{
  config,
  lib,
  ...
}:
{
  config = lib.mkIf (config.nixos.desktop.enable && config.nixos.desktop.displayManager == "gdm") {
    services.displayManager.gdm.enable = true;
  };
}
