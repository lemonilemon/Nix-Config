{
  ...
}:
{
  home.desktop.hyprland.eww.enable = true;
  home.desktop.hyprland.awww.enable = true;

  nixos.desktop.gnome.enable = false;

  # Trying Limine ("grub" is the other themed loader; the boot order follows
  # this choice on activation).
  nixos.desktop.bootloader = "limine";
}
