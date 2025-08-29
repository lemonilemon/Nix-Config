{ config, lib, pkgs, platformConfig, ... }: {
  imports = [
    # Import hardware configuration specific to this host
    ./hardware-configuration.nix
    
    # Note: WSL doesn't need boot.nix, sound.nix, i18n.nix, or nvidia.nix
    # WSL handles these through Windows host
  ];

  # Host-specific settings
  networking.hostName = "SpaceNix-WSL";
  
  # Basic locale settings for WSL (no input method needed)
  i18n.defaultLocale = "en_US.UTF-8";
  i18n.supportedLocales = [
    "en_US.UTF-8/UTF-8"
    "zh_TW.UTF-8/UTF-8"
  ];
  
  # Common settings
  system.stateVersion = "25.05";
  time.timeZone = "Asia/Taipei";
  nixos.desktop.gnome.enable = lib.mkDefault false;
}