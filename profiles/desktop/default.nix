# Edit this configuration file to define what should be installed on
# your system. Help is available in the configuration.nix(5) man page, on
# https://search.nixos.org/options and in the NixOS manual (`nixos-help`).

{
  pkgs,
  ...
}:
{
  imports = [
    # Include the results of the hardware scan.
    ./hardware-configuration.nix
    ./config.nix
    ./nvidia.nix # For NVIDIA graphics cards
    ../base.nix
    ../i18n.nix
    ../boot.nix
    ../sound.nix
  ];

  formFactor = "desktop";

  nixos.general.firewall.trustedSubnets = [ "192.168.0.0/24" ];

  environment.sessionVariables = {
    NIXHOST = "desktop";
  };

  # Set your time zone.
  time.timeZone = "Asia/Taipei";

  environment.systemPackages = with pkgs; [
    ntfs3g # NTFS driver
    pciutils # Inspecting PCI devices
    mesa # 3D graphics library
    mesa-demos # Test utilities for OpenGL
  ];

  hardware = {
    graphics.enable = true;
    bluetooth.enable = true;
  };
  services.blueman.enable = true;

  security.polkit.enable = true;

  system.stateVersion = "25.11";

}
