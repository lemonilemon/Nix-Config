{
  lib,
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
    package = inputs.rose-pine-hyprcursor.packages.${pkgs.system}.default;
    hyprcursor = {
      enable = true;
      size = 24;
    };
  };
  gtk = {
    enable = true;
    # iconTheme = {
    #   name = lib.mkForce "rose-pine";
    #   package = pkgs.rose-pine-icon-theme;
    # };
    # theme = {
    #   name = lib.mkForce "rose-pine";
    #   package = pkgs.rose-pine-gtk-theme;
    # };
    cursorTheme = {
      name = lib.mkForce "BreezeX-RosePine-Linux";
      package = pkgs.rose-pine-cursor;
      size = 24;
    };
  };
}
