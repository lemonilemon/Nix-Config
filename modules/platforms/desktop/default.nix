{ config, lib, pkgs, platformConfig, ... }: 
lib.mkIf platformConfig.isDesktop {
  # Desktop-specific optimizations
  services.xserver.videoDrivers = [ "nvidia" ]; # Adjust based on your hardware
  
  # Performance settings for desktop
  services.thermald.enable = false; # Usually not needed on desktop
  
  # Desktop hardware optimizations
  hardware.bluetooth.enable = true;
  services.blueman.enable = true;

  # Gaming and multimedia packages for desktop
  environment.systemPackages = with pkgs; [
    # Add desktop-specific packages here if needed
  ];
}