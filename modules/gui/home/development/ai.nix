{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.gui.development.opencode.enable {
    home.packages = with pkgs; [
      opencode-desktop
    ];
  };
}
