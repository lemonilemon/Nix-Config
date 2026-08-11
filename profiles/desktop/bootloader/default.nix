{
  lib,
  config,
  pkgs,
  ...
}:
let
  cfg = config.nixos.desktop.bootloader;

  # The ESP paths the firmware entries point at, as efibootmgr prints them.
  loaderPaths = {
    grub = "\\EFI\\NIXOS-BOOT\\GRUBX64.EFI";
    limine = "\\EFI\\LIMINE\\BOOTX64.EFI";
  };
  loaderPath = loaderPaths.${cfg};
  benchedPath = loaderPaths.${if cfg == "grub" then "limine" else "grub"};
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
      buildable; the inactive one stays benched on the ESP and switching is
      changing this value plus one rebuild -- including which loader the
      firmware starts, which activation keeps in sync.
    '';
  };

  config = {
    # Make the option own what the firmware boots, so a rebuild is the whole
    # switch. This board REGENERATES BootOrder from its own priority store on
    # every boot (verified: a written order came back as the firmware's full
    # original list), so reordering cannot stick. Instead, the firmware's
    # favorite entry is retargeted at the active loader's binary -- the
    # firmware keeps booting "its" first choice; we control what it means.
    # Idempotent, loud on every outcome, and inert when EFI variables are
    # unavailable (installer, nixos-enter) or the entries look unfamiliar.
    system.activationScripts.bootloaderOrder = ''
      # Known limitation (review finding): activation also fires on
      # `nixos-rebuild test`, where bootloader files are NOT installed, so a
      # retarget there points NVRAM at a loader state one switch ahead. On
      # this host the firmware ignores NVRAM anyway (the cached-path copy is
      # the real mechanism); accept the window rather than grow machinery.
      if [ -d /sys/firmware/efi/efivars ] && [ -z "''${IN_NIXOS_ENTER:-}" ]; then
        if state=$(${pkgs.efibootmgr}/bin/efibootmgr 2>/dev/null); then
          order=$(printf '%s\n' "$state" | ${pkgs.gnused}/bin/sed -n 's/^BootOrder: //p')
          top=''${order%%,*}
          topline=""
          if [ -n "$top" ]; then
            topline=$(printf '%s\n' "$state" | ${pkgs.gnugrep}/bin/grep -i "^Boot$top" || true)
          fi
          if [ -z "$topline" ]; then
            echo "bootloaderOrder: no readable BootOrder/top entry; leaving NVRAM alone" >&2
          elif printf '%s\n' "$topline" | ${pkgs.gnugrep}/bin/grep -qiF '${loaderPath}'; then
            echo "bootloaderOrder: Boot$top already boots the active loader"
          elif printf '%s\n' "$topline" | ${pkgs.gnugrep}/bin/grep -qiF '${benchedPath}'; then
            label=$(printf '%s\n' "$topline" \
              | ${pkgs.gnused}/bin/sed -e 's/^Boot[0-9A-Fa-f]\{4\}\*\{0,1\} //' -e 's/\t.*//')
            espdev=$(${pkgs.util-linux}/bin/findmnt -no SOURCE /boot || true)
            partn=$(${pkgs.util-linux}/bin/lsblk -no PARTN "$espdev" 2>/dev/null | ${pkgs.coreutils}/bin/head -n1)
            pkname=$(${pkgs.util-linux}/bin/lsblk -no PKNAME "$espdev" 2>/dev/null | ${pkgs.coreutils}/bin/head -n1)
            # Validate BEFORE the destructive delete: a failed create after
            # -B would leave the firmware's favorite entry gone.
            if [ -z "$espdev" ] || [ -z "$partn" ] || [ -z "$pkname" ]; then
              echo "bootloaderOrder: cannot resolve ESP disk/partition; leaving NVRAM alone" >&2
            elif out=$(${pkgs.efibootmgr}/bin/efibootmgr -b "$top" -B 2>&1) \
              && out=$(${pkgs.efibootmgr}/bin/efibootmgr -c -b "$top" -d "/dev/$pkname" -p "$partn" \
                         -L "$label" -l '${loaderPath}' 2>&1); then
              echo "bootloaderOrder: retargeted Boot$top ($label) -> ${loaderPath}"
            else
              echo "bootloaderOrder: FAILED retargeting Boot$top: $out" >&2
            fi
          else
            # Top entry is not one of our loaders (unexpected on this host):
            # fall back to a plain reorder for firmware that respects it.
            entry=$(printf '%s\n' "$state" \
              | ${pkgs.gnugrep}/bin/grep -iF '${loaderPath}' \
              | ${pkgs.gnugrep}/bin/grep -oE '^Boot[0-9A-Fa-f]{4}' \
              | ${pkgs.coreutils}/bin/head -n1 | ${pkgs.coreutils}/bin/cut -c5-8)
            if [ -z "$entry" ]; then
              echo "bootloaderOrder: no NVRAM entry for ${loaderPath}; leaving NVRAM alone" >&2
            else
              rest=$(printf '%s\n' "$order" | ${pkgs.coreutils}/bin/tr ',' '\n' \
                | ${pkgs.gnugrep}/bin/grep -vx "$entry" | ${pkgs.coreutils}/bin/paste -sd, -)
              if out=$(${pkgs.efibootmgr}/bin/efibootmgr -o "$entry''${rest:+,$rest}" 2>&1); then
                echo "bootloaderOrder: BootOrder set to $entry''${rest:+,$rest}"
              else
                echo "bootloaderOrder: efibootmgr -o FAILED, BootOrder unchanged: $out" >&2
              fi
            fi
          fi
        fi
      fi
    '';
  };
}
