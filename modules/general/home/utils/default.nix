{
  lib,
  pkgs,
  config,
  ...
}:
{
  config = lib.mkIf config.home.general.utils.enable {
    home.packages = with pkgs; [
      just
      mosh
      sshfs
      killall
    ];
  };
}
