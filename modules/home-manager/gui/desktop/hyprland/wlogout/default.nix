{
  username,
  ...
}:
{
  # Icons
  home.file.".config/wlogout/icons" = {
    enable = true;
    source = ./icons;
    recursive = true;
  };
  # Logout menu
  programs.wlogout = {
    enable = true;
  };
}
