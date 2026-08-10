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
      buildable; the inactive one stays benched on the ESP (its NVRAM entry
      survives) and switching is changing this value plus one rebuild.
    '';
  };
}
