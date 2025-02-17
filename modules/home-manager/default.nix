{
  lib,
  username,
  ...
}:
{
  home.username = "${username}";
  home.homeDirectory = "/home/${username}";
  imports = [
    ./cli
    ./gui
    ./general
  ];
  home.stateVersion = "24.11";
  programs.home-manager.enable = true;
  general-settings.enable = lib.mkDefault true;
  cli-settings.enable = lib.mkDefault true;
}
