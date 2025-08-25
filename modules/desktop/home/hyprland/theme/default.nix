{
  lib,
  pkgs,
  inputs,
  ...
}:
{
  catppuccin.kvantum.enable = false;
  catppuccin.gtk.icon.enable = false;
  home.packages = with pkgs; [
    rose-pine-cursor
  ];
  home.pointerCursor = {
    enable = true;
    name = "rose-pine-hyprcursor";
    package = inputs.rose-pine-hyprcursor.packages.${pkgs.system}.default;
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
      name = lib.mkForce "rose-pine";
      package = pkgs.rose-pine-gtk-theme;
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

    gtk4.extraConfig = {
      gtk-application-prefer-dark-theme = 1;
    };
  };

  home.sessionVariables = {
    GTK_THEME = "rose-pine";
  };

  qt = {
    enable = true;
    platformTheme.name = "gtk3";
    style.name = "gtk2";
  };
}
