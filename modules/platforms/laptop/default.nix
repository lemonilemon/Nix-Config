{ config, lib, pkgs, platformConfig, ... }: 
lib.mkIf platformConfig.isLaptop {
  # Laptop-specific hardware and power management
  services.tlp = {
    enable = true;
    settings = {
      CPU_SCALING_GOVERNOR_ON_AC = "performance";
      CPU_SCALING_GOVERNOR_ON_BAT = "powersave";
      START_CHARGE_THRESH_BAT0 = 20;
      STOP_CHARGE_THRESH_BAT0 = 80;
    };
  };

  # Power management services
  services.thermald.enable = true;
  services.acpid.enable = true;
  services.upower.enable = true;
  services.power-profiles-daemon.enable = false;
  
  # Auto CPU frequency management
  services.auto-cpufreq = {
    enable = true;
    settings = {
      battery = {
        governor = "powersave";
        turbo = "never";
      };
      charger = {
        governor = "performance";
        turbo = "auto";
      };
    };
  };

  # Power management configuration
  powerManagement = {
    enable = true;
    powertop.enable = true;
    cpuFreqGovernor = "powersave";
  };

  programs.light.enable = true;

  # Laptop-specific packages
  environment.systemPackages = with pkgs; [
    powertop
    acpi
    brightnessctl
    ntfs3g       # NTFS driver for external drives
    pciutils     # Inspecting PCI devices
  ];

  # Laptop hardware optimizations
  hardware = {
    bluetooth.enable = true;
    graphics.enable = true;
    enableRedistributableFirmware = true;
    
    # NVIDIA settings for laptops (if present)
    nvidia = lib.mkIf config.hardware.nvidia.modesetting.enable {
      prime.sync.enable = lib.mkDefault true;
      powerManagement.enable = true;
    };
  };
  
  # Bluetooth management
  services.blueman.enable = true;

  # Networking configuration for laptops
  networking = {
    wireless.enable = false; # Use NetworkManager instead
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
    NIXHOST = "laptop";
  };

  # Security
  security.polkit.enable = true;
}