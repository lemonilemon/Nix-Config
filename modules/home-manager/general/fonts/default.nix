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
        emoji = [ "Noto Emoji" ];
      };
    };
    home.packages = with pkgs; [
      noto-fonts
      noto-fonts-cjk-sans
      noto-fonts-emoji
      nerd-fonts.fira-code
    ];
  };
}
