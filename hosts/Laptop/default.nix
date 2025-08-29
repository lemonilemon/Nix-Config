{ config, lib, pkgs, platformConfig, ... }: {
  imports = [
    # Import hardware configuration specific to this host
    ./hardware-configuration.nix
    
    # Import common modules
    ../common/boot.nix
    ../common/sound.nix
    ../common/i18n.nix
  ] ++ lib.optionals platformConfig.isDesktop [
    ../common/nvidia.nix
  ];

  # Host-specific settings
  networking.hostName = "SpaceNix-Laptop";
  
  # Common settings
  system.stateVersion = "25.05";
  time.timeZone = "Asia/Taipei";
  nixos.desktop.gnome.enable = lib.mkDefault false;
}