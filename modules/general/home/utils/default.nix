{
  lib,
  pkgs,
  config,
  ...
}:
{
  config = lib.mkIf config.home.general-settings.utils.enable {
    home.packages = with pkgs; [
      just
      autossh
      sshfs
      killall
    ];
  };
}
