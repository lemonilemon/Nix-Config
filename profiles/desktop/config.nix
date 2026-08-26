{
  ...
}:
{
  home.desktop.hyprland.eww.enable = true;
  home.desktop.hyprland.awww.enable = true;

  nixos.desktop.gnome.enable = false;

  nixos.desktop.bootloader = "limine";
  # This board ignores NVRAM retargeting and boots its cached NixOS-boot path
  # (verified 2026-08-11), so the active loader has to be installed there.
  nixos.desktop.bootloaderFirmwareIgnoresNvram = true;
}
