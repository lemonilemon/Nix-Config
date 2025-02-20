{
  lib,
  username,
  ...
}:
{
  imports = [
    ./cli
    ./gui
    ./general
  ];
  config = {
    home.username = "${username}";
    home.homeDirectory = "/home/${username}";
    home.stateVersion = "24.11";
    programs.home-manager.enable = true;
    cli-settings.enable = lib.mkDefault true;
    gui-settings.enable = lib.mkDefault true;
    general-settings.enable = lib.mkDefault true;
  };
}
