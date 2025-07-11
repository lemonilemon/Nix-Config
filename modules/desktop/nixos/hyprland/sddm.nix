{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.nixos.desktop.hyprland.enable {
    services.displayManager.sddm = {
      enable = true;
      wayland = {
        enable = true;
        compositor = "weston";
      };
      autoNumlock = true;
      package = pkgs.kdePackages.sddm;
      enableHidpi = true;
      theme = "sddm-astronaut-theme";
      settings = {
        Theme = {
          CursorTheme = "rose-pine";
          CursorSize = 24;
        };
      };
      extraPackages = with pkgs; [
        kdePackages.qtsvg
        kdePackages.qtvirtualkeyboard
        kdePackages.qtmultimedia
      ];
    };
    environment.systemPackages = with pkgs; [
      sddm-astronaut
      rose-pine-cursor
    ];
    catppuccin.sddm.enable = lib.mkDefault false;
  };
}
