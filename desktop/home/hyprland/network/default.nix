{
  pkgs,
  ...
}:
{
  home.packages = with pkgs; [
    networkmanagerapplet
    hicolor-icon-theme
    adwaita-icon-theme
    gtk3
    libappindicator
    stalonetray
    killall
  ];
}
