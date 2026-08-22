{ lib, ... }:
{
  imports = [
    ./grub.nix
    ./limine.nix
  ];

  options.nixos.desktop.bootloader = lib.mkOption {
    type = lib.types.enum [
      "grub"
      "limine"
    ];
    default = "grub";
    description = ''
      Which SpaceNix-themed bootloader drives this machine. Both are kept
      buildable; the inactive one stays benched on the ESP and switching is
      changing this value plus one rebuild. The selected loader is installed
      at the EFI path already cached by the firmware.
    '';
  };
}
