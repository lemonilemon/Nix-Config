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

  services.thermald.enable = true;
  services.acpid.enable = true;
  programs.light.enable = true;

  # Power management packages
  environment.systemPackages = with pkgs; [
    powertop
    acpi
    brightnessctl
  ];

  # Laptop hardware optimizations
  hardware.bluetooth.enable = true;
  services.blueman.enable = true;
}