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

  nixpkgs.config = {
    allowUnfree = true;
  };

  time.timeZone = "Asia/Taipei";
  system.stateVersion = "25.11";
}
