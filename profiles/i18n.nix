{ pkgs, ... }:
{
  i18n = {
    defaultLocale = "en_US.UTF-8";
    supportedLocales = [
      "en_US.UTF-8/UTF-8"
      "zh_TW.UTF-8/UTF-8"
    ];
    inputMethod = {
      enable = true;
      type = "fcitx5";
      fcitx5 = {
        waylandFrontend = true;
        addons = with pkgs; [
          fcitx5-rime
          rime-data
          fcitx5-material-color
          fcitx5-chinese-addons
        ];
        settings = {
          inputMethod = {
            "Groups/0" = {
              Name = "Default";
              "Default Layout" = "us";
              DefaultIM = "keyboard-us";
            };
            "Groups/0/Items/0".Name = "keyboard-us";
            "Groups/0/Items/1".Name = "rime";
            GroupOrder."0" = "Default";
          };
        };
      };
    };
  };
}
