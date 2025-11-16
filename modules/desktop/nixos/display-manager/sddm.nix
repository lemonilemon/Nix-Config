{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf (config.nixos.desktop.enable && config.nixos.desktop.displayManager == "sddm") {
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
          CursorTheme = "BreezeX-RosePine-Linux";
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
