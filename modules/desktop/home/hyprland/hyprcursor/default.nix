{
  pkgs,
  inputs,
  ...
}:
{
  home.packages = with pkgs; [
    rose-pine-cursor
  ];
  home.pointerCursor = {
    enable = true;
    name = "rose-pine-hyprcursor";
    size = 32;
    package = inputs.rose-pine-hyprcursor.packages.${pkgs.system}.default;
    hyprcursor = {
      enable = true;
      size = 32;
    };
  };
  wayland.windowManager.hyprland = {
    settings = {
      exec-once = [
        "gsettings set org.gnome.desktop.interface cursor-theme BreezeX-RosePine-Linux"
        "gsettings set org.gnome.desktop.interface cursor-size 32"
      ];
    };
  };
  gtk = {
    enable = true;
    iconTheme = {
      name = "rose-pine";
      package = pkgs.rose-pine-icon-theme;
    };
    theme = {
      name = "rose-pine";
      package = pkgs.rose-pine-gtk-theme;
    };
    cursorTheme = {
      name = "rose-pine-cursor";
      package = pkgs.rose-pine-cursor;
      size = 32;
    };
  };
}
