{
  lib,
  pkgs,
  config,
  ...
}:
{
  config = lib.mkIf config.home.general-settings.fonts.enable {
    fonts.fontconfig = {
      enable = true;
      # defaultFonts = {
      #   sansSerif = [
      #     "Noto Sans CJK TC"
      #     "Noto Sans CJK SC"
      #     "Noto Sans"
      #     "DejaVu Sans"
      #   ];
      #   serif = [
      #     "Noto Serif CJK TC"
      #     "Noto Serif CJK SC"
      #     "Noto Serif"
      #     "DejaVu Sans"
      #   ];
      #   emoji = [
      #     "Noto Color Emoji"
      #     "Noto Emoji"
      #     "Awesome Font"
      #   ];
      # };
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
