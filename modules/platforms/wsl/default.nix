{ config, lib, pkgs, username, platformConfig, ... }: 
lib.mkIf platformConfig.isWSL {
  # WSL-specific configuration
  wsl = {
    enable = true;
    defaultUser = username;
    startMenuLaunchers = true;
    docker-desktop.enable = true;
    wslConf = {
      automount = {
        enabled = true;
        root = "/mnt";
        options = "metadata,uid=1000,gid=100";
      };
      network.generateHosts = false;
      interop.appendWindowsPath = true;
    };
  };

  # WSL-specific packages
  environment.systemPackages = with pkgs; [
    wsl-open
  ];

  # Environment variables
  environment.sessionVariables = {
    NIXHOST = "NixOS-wsl";
  };

  # Disable services inappropriate for WSL
  services = {
    xserver.enable = lib.mkForce false;
    tlp.enable = lib.mkForce false;
    thermald.enable = lib.mkForce false;
    blueman.enable = lib.mkForce false;
    power-profiles-daemon.enable = lib.mkForce false;
    auto-cpufreq.enable = lib.mkForce false;
    upower.enable = lib.mkForce false;
  };

  # WSL-optimized networking
  networking = {
    dhcpcd.enable = lib.mkForce false;
    useNetworkd = lib.mkForce false;
    networkmanager.enable = lib.mkForce false;
  };

  # Disable hardware-specific options
  hardware = {
    bluetooth.enable = lib.mkForce false;
    graphics.enable = lib.mkForce false;
    nvidia.modesetting.enable = lib.mkForce false;
  };

  # Disable power management for WSL
  powerManagement.enable = lib.mkForce false;
}