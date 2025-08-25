{ config, lib, pkgs, platformConfig, ... }: 
lib.mkIf platformConfig.isWSL {
  # WSL-specific configuration
  wsl = {
    enable = true;
    defaultUser = "lemonilemon";
    startMenuLaunchers = false;
    wslConf = {
      automount = {
        enabled = true;
        root = "/mnt";
        options = "metadata,uid=1000,gid=100";
      };
      network.generateHosts = true;
    };
  };

  # Disable services inappropriate for WSL
  services = {
    xserver.enable = lib.mkForce false;
    tlp.enable = lib.mkForce false;
    thermald.enable = lib.mkForce false;
    blueman.enable = lib.mkForce false;
  };

  # WSL-optimized networking
  networking = {
    dhcpcd.enable = lib.mkForce false;
    useNetworkd = lib.mkForce false;
  };

  # Disable hardware-specific options
  hardware.bluetooth.enable = lib.mkForce false;
}