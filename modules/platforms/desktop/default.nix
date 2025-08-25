{ config, lib, pkgs, platformConfig, ... }: 
lib.mkIf platformConfig.isDesktop {
  # Desktop-specific configuration
  
  # High-performance power settings
  powerManagement = {
    enable = true;
    cpuFreqGovernor = "performance";
  };

  # Desktop hardware optimization
  hardware = {
    graphics.enable = true;
    enableRedistributableFirmware = true;
    bluetooth.enable = true;
  };

  # Desktop-specific packages
  environment.systemPackages = with pkgs; [
    pciutils     # Inspecting PCI devices
    mesa         # 3D graphics library
    glxinfo      # Test utilities for OpenGL
    vulkan-tools # Vulkan utilities
  ];

  # Desktop services
  services = {
    blueman.enable = true;
    # Disable power management services not needed for desktop
    tlp.enable = lib.mkForce false;
    auto-cpufreq.enable = lib.mkForce false;
    thermald.enable = false; # Usually not needed on desktop
  };

  # Networking configuration for desktop
  networking = {
    wireless.enable = false; # Use NetworkManager
    networkmanager = {
      enable = true;
      plugins = with pkgs; [
        networkmanager-openvpn
        networkmanager-openconnect
      ];
    };
    useDHCP = false;
  };

  # Environment variables
  environment.sessionVariables = {
    NIXHOST = "desktop";
  };

  # Security
  security.polkit.enable = true;
}