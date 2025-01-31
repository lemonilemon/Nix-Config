{
  username,
  ...
}:
{
  home.username = "${username}";
  home.homeDirectory = "/home/${username}";
  imports = [
    ./cli
    ./general
  ];
  home.stateVersion = "24.11";
  programs.home-manager.enable = true;
  general-settings.enable = true;
  cli-settings.enable = true;
}
