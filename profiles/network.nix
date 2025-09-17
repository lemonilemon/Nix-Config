{
  pkgs,
  lib,
  ...
}:
{
  networking.wireless.enable = lib.mkDefault false; # explicitly disable wireless
  networking.networkmanager = {
    enable = lib.mkDefault true; # Easiest to use and most distros use this by default.
    plugins =
      with pkgs;
      lib.mkDefault [
        networkmanager-openvpn # OpenVPN support
        networkmanager-openconnect # OpenConnect support
      ];
  };
  hardware.enableRedistributableFirmware = lib.mkDefault true; # For wifi firmware

}
