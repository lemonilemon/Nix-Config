{ config, lib, pkgs, platformConfig, ... }: {
  imports = [
    # Import appropriate hardware and boot configuration based on platform
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
  
  # Import platform-agnostic host settings if they exist
  nixos.desktop.gnome.enable = false;
}