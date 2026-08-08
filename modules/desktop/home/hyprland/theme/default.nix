{
  lib,
  config,
  pkgs,
  inputs,
  ...
}:
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    catppuccin.hyprland.enable = false; # incompatible with configType = "hyprlang"
    catppuccin.kvantum.enable = false;
    catppuccin.gtk.icon.enable = false;
    home.packages = with pkgs; [
      rose-pine-cursor
    ];
    home.pointerCursor = {
      enable = true;
      name = "rose-pine-hyprcursor";
      package = inputs.rose-pine-hyprcursor.packages.${pkgs.stdenv.hostPlatform.system}.default;
      hyprcursor = {
        enable = true;
        size = 24;
      };
    };
    gtk = {
      enable = true;
      iconTheme = {
        name = lib.mkForce "rose-pine";
        package = pkgs.rose-pine-icon-theme;
      };
      theme = {
        name = lib.mkForce "catppuccin-${config.catppuccin.flavor}-${config.catppuccin.accent}-standard";
        package = pkgs.catppuccin-gtk.override {
          variant = config.catppuccin.flavor;
          accents = [ config.catppuccin.accent ];
        };
      };
      cursorTheme = {
        name = lib.mkForce "BreezeX-RosePine-Linux";
        package = pkgs.rose-pine-cursor;
        size = 24;
      };
      # Additional GTK settings for dark theme preference

      gtk3.extraConfig = {
        gtk-application-prefer-dark-theme = 1;
      };

      gtk4 = {
        theme = config.gtk.theme;
        extraConfig = {
          gtk-application-prefer-dark-theme = 1;
        };
      };
    };

    home.sessionVariables = {
      GTK_THEME = config.gtk.theme.name;
    };

    # The xdg-desktop-portal Settings backend serves this to every
    # portal-aware app (zen, Electron); without it they assume light. The
    # gtk-application-prefer-dark-theme keys above only reach plain GTK3.
    dconf.settings."org/gnome/desktop/interface".color-scheme = "prefer-dark";

    qt = {
      enable = true;
      platformTheme.name = "gtk3";
      style.name = "gtk2";
    };
  };
}
