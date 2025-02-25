{
  pkgs,
  ...
}:
{
  home.packages = with pkgs; [
    webcord
    _1password-gui
  ];
}
