{
  lib,
  username,
  pkgs,
  hostname,
  ...
}:
{
  # Basic hardware and system settings
  hardware.graphics.enable32Bit = lib.mkDefault true;
  services.dbus.implementation = lib.mkDefault "broker";
  networking.hostName = lib.mkDefault hostname;

  nixos.general.network.dns.servers = [
    "8.8.8.8#dns.google"
    "8.8.4.4#dns.google"
  ];

  nixpkgs.config = {
    allowUnfree = true;
  };

  time.timeZone = "Asia/Taipei";
  system.stateVersion = "25.11";
}
