{
  lib,
  config,
  pkgs,
  ...
}:
let
  cfg = config.nixos.desktop.bootloader;

  # The ESP paths the firmware entries point at, as efibootmgr prints them.
  # If a loader's install id ever changes, the activation step logs a warning
  # and touches nothing -- entry CREATION stays the installers' job, only the
  # ORDER is owned here.
  loaderPath =
    {
      grub = "\\EFI\\NIXOS-BOOT\\GRUBX64.EFI";
      limine = "\\EFI\\LIMINE\\BOOTX64.EFI";
    }
    .${cfg};
in
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
      survives) and switching is changing this value plus one rebuild --
      including the firmware boot order, which activation keeps in sync.
    '';
  };

  config = {
    # Make the option own the firmware boot order too, so a rebuild is the
    # whole switch. Idempotent: reads first, rewrites BootOrder only when the
    # active loader is not already on top, and degrades to a warning when the
    # expected entry is missing (fresh ESP, renamed id) or EFI variables are
    # unavailable (installer, nixos-enter).
    system.activationScripts.bootloaderOrder = ''
      if [ -d /sys/firmware/efi/efivars ]; then
        if state=$(${pkgs.efibootmgr}/bin/efibootmgr 2>/dev/null); then
          entry=$(printf '%s\n' "$state" \
            | ${pkgs.gnugrep}/bin/grep -iF '${loaderPath}' \
            | ${pkgs.gnugrep}/bin/grep -oE '^Boot[0-9A-Fa-f]{4}' \
            | ${pkgs.coreutils}/bin/head -n1 | ${pkgs.coreutils}/bin/cut -c5-8)
          order=$(printf '%s\n' "$state" | ${pkgs.gnused}/bin/sed -n 's/^BootOrder: //p')
          if [ -z "$entry" ]; then
            echo "bootloaderOrder: no NVRAM entry for ${loaderPath}; leaving BootOrder alone" >&2
          elif [ "''${order%%,*}" != "$entry" ]; then
            rest=$(printf '%s\n' "$order" | ${pkgs.coreutils}/bin/tr ',' '\n' \
              | ${pkgs.gnugrep}/bin/grep -vx "$entry" | ${pkgs.coreutils}/bin/paste -sd, -)
            ${pkgs.efibootmgr}/bin/efibootmgr -o "$entry''${rest:+,$rest}" >/dev/null \
              || echo "bootloaderOrder: efibootmgr -o failed; BootOrder unchanged" >&2
          fi
        fi
      fi
    '';
  };
}
