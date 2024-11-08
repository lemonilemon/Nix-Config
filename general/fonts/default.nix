{
  pkgs,
  ...
}:
{
  fonts.packages = with pkgs; [
    noto-fonts
    noto-fonts-cjk-sans
    noto-fonts-emoji
    (nerdfonts.override { fonts = [ "FiraCode" ]; })
  ];
  fonts.fontDir.enable = true;
}
