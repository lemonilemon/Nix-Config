{
  ...
}:
{
  home.desktop.hyprland.eww.enable = true;
  home.desktop.hyprland.awww.enable = true;

  nixos.desktop.gnome.enable = false;

  nixos.desktop.bootloader = "limine";

  # This panel is 1920x1200 (AU Optronics 0x339F, eDP-1); the option defaults
  # to the desktop's 1920x1080.
  nixos.desktop.bootloaderResolution = "1920x1200";

  # Verified 2026-08-27: with Limine leading BootOrder, the firmware still
  # demoted it during boot and loaded the cached NixOS-boot path.
  nixos.desktop.bootloaderFirmwareIgnoresNvram = true;
}
