{
  lib,
  pkgs,
  config,
  ...
}:
{
  options = {
    general-settings.fonts = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.general-settings.enable;
        description = "Enable my font settings";
      };
    };
  };
  config = lib.mkIf config.general-settings.fonts.enable {
    fonts.fontconfig = {
      enable = true;
      defaultFonts = {
        emoji = [
          "Noto Color Emoji"
          "Noto Emoji"
          "Awesome Font"
        ];
      };
    };
    home.packages = with pkgs; [
      noto-fonts
      noto-fonts-cjk-sans
      noto-fonts-emoji
      noto-fonts-color-emoji
      nerd-fonts.fira-code
      nerd-fonts.jetbrains-mono
      font-awesome
    ];
  };
}
