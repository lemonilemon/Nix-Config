{ config, lib, pkgs, platformConfig, ... }: {
  imports = [
    # Import hardware and boot configuration based on platform
    ./boot.nix
    ./sound.nix
    ./i18n.nix
  ] ++ lib.optionals platformConfig.isLaptop [
    ../../profiles/laptop/hardware-configuration.nix
  ] ++ lib.optionals platformConfig.isDesktop [
    ../../profiles/desktop/hardware-configuration.nix
    ./nvidia.nix
  ];

  # Host-specific settings
  networking.hostName = "SpaceNix";
  
  # Set system state version
  system.stateVersion = "25.05";
  
  # Set time zone for this host
  time.timeZone = "Asia/Taipei";
  
  # Disable GNOME (using Hyprland instead)
  nixos.desktop.gnome.enable = lib.mkDefault false;
}