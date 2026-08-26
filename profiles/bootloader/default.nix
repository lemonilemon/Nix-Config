{ config, lib, ... }:
{
  imports = [
    ./grub.nix
    ./limine.nix
  ];

  # NVRAM writes on such a board are wasted (it rewrites the order at boot)
  # and the limine installer crashes re-writing its dangling BootOrder refs,
  # so stop managing NVRAM entirely; the cached-path overwrite is what boots.
  config.boot.loader.efi.canTouchEfiVariables =
    lib.mkIf config.nixos.desktop.bootloaderFirmwareIgnoresNvram (lib.mkForce false);

  options.nixos.desktop = {
    bootloader = lib.mkOption {
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

    bootloaderResolution = lib.mkOption {
      type = lib.types.strMatching "[0-9]+x[0-9]+";
      default = "1920x1080";
      example = "1920x1200";
      description = ''
        This host's native panel mode. Not cosmetic: it sizes the generated
        theme assets, and it is the mode the menu hands to Plymouth. Get it
        wrong and the menu letterboxes AND the kernel has to renegotiate the
        framebuffer from EDID, which is the flash between menu and splash.

        Both loaders read it, so the menu mode and the kernel-entry mode
        cannot drift apart.
      '';
    };

    bootloaderFirmwareIgnoresNvram = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = ''
        Set when this board boots its NixOS-boot option from a CACHED
        definition rather than from NVRAM -- verified on the desktop board
        2026-08-11 by retargeting the NVRAM entry to Limine and watching the
        firmware load GRUB from the old path anyway. On such a board NVRAM is
        a suggestion box: the file at the cached path is the only thing it
        honors, so the active loader's binary has to be placed AT that path
        (see the limine branch's extraInstallCommands).

        Leave false on firmware that behaves. The workaround overwrites the
        binary behind the machine's only Linux boot entry, which is not
        something to inherit on spec. If a host boots the old loader even
        though Limine leads BootOrder, that IS this quirk, and this is the
        fix. Setting it also forces canTouchEfiVariables off, so nothing
        manages NVRAM at all on such a board.
      '';
    };
  };
}
