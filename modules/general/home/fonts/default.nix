{
  lib,
  pkgs,
  config,
  ...
}:
{
  config = lib.mkIf config.home.general.fonts.enable {
    fonts.fontconfig = {
      enable = true;
    };
    home.packages = with pkgs; [
      corefonts # Arial, Times New Roman, Courier New, etc.
      cm_unicode
      ubuntu-classic
      source-han-serif
      noto-fonts
      noto-fonts-cjk-sans
      noto-fonts-cjk-serif
      noto-fonts-color-emoji
      nerd-fonts.fira-code
      nerd-fonts.jetbrains-mono
      nerd-fonts.meslo-lg
      dejavu_fonts
      font-awesome
    ];
  };
}
