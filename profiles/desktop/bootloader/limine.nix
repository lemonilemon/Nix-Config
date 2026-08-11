{
  pkgs,
  config,
  lib,
  ...
}:
let
  jbm = "${pkgs.jetbrains-mono}/share/fonts/truetype";

  # Same art as the GRUB branch -- identical expression, one store path.
  wallpaper = pkgs.runCommand "spacenix-boot-wallpaper" {
    nativeBuildInputs = [ (pkgs.python3.withPackages (ps: [ ps.pillow ])) ];
  } ''python3 ${../boot-theme/wallpaper.py} "$out" ${jbm}/JetBrainsMono-SemiBold.ttf'';

  # Menu font: Spleen 8x16 converted to Limine's raw CP437 format, with the
  # required menu glyphs verified present (Tamsyn, for one, lacks the arrows).
  terminalFont = pkgs.runCommand "limine-term-font-spleen" {
    nativeBuildInputs = [ pkgs.python3 ];
  } ''python3 ${./font.py} ${pkgs.spleen}/share/consolefonts/spleen-8x16.psfu "$out"'';
in
{
  config = lib.mkIf (config.nixos.desktop.bootloader == "limine") {
    boot.loader.grub.enable = false;
    boot.loader.grub2-theme.enable = false;

    # Deliberately NOT catppuccin.limine: its theme sets an opaque
    # term_background that would defeat the translucent panel below. (An
    # earlier claim here that its theme path is broken at the pinned revs
    # was refuted by review -- the port evaluates fine; the opacity conflict
    # is the real reason.) The style IS catppuccin-mocha, applied directly.
    catppuccin.limine.enable = lib.mkForce false;

    boot.loader.limine = {
      enable = true;
      maxGenerations = 5; # matches the GRUB configurationLimit
      # Keep the kernel-entry mode identical to the menu mode so the
      # menu -> Plymouth handoff does not renegotiate from EDID (the grub
      # branch gets the same via gfxpayloadEfi = "keep").
      resolution = "1920x1080";
      style = {
        wallpapers = [ wallpaper ];
        backdrop = "1e1e2e";
        interface = {
          # Booting via the firmware's cached entry leaves the GOP in a state
          # the default probe cannot use -- gterm_init failed and Limine fell
          # back to its unstyled text terminal (observed 2026-08-11: entries
          # fine, no wallpaper/font/colors). An explicit resolution makes
          # fb_init walk the mode list and SetMode assertively instead of
          # trusting the inherited mode.
          resolution = "1920x1080";
          # The wordmark in the wallpaper is the branding; this suppresses
          # Limine's own line (confirmed rendering as no title).
          branding = "";
          brandingColor = "cba6f7"; # mauve
          helpColor = "6c7086"; # overlay0
          helpColorBright = "a6e3a1"; # green: the autoboot countdown digit
        };
        graphicalTerminal = {
          # mocha ANSI palette, as in catppuccin/limine's mocha port
          palette = "1e1e2e;f38ba8;a6e3a1;f9e2af;89b4fa;f5c2e7;94e2d5;cdd6f4";
          brightPalette = "585b70;f38ba8;a6e3a1;f9e2af;89b4fa;f5c2e7;94e2d5;cdd6f4";
          foreground = "cdd6f4";
          # TT in TTRRGGBB is TRANSPARENCY (review finding): 0x37 = 78%
          # opaque, matching the mockups. The earlier C7 had it inverted and
          # rendered a nearly invisible panel.
          background = "371E1E2E";
          # The menu floats as a card under the wordmark. Margin 240 keeps
          # the wordmark band clear AND fits the whole tree: at 2x scale the
          # terminal gets (1080-480)/32 = 18 rows, minus 8 rows of menu
          # chrome = 10 for the tree of 7 (dir + 5 generations + Windows).
          # The previous 320 left 5 rows and scrolled Windows off-screen.
          margin = 240;
          marginGradient = 24;
          font.scale = "2x2";
        };
      };
      additionalFiles."limine/term-font.bin" = terminalFont;
      # term_font has no typed option; extraConfig is prepended to limine.conf.
      extraConfig = ''
        term_font: boot():/limine/term-font.bin
        term_font_size: 8x16
      '';
      # Limine has no os-prober; Windows is pinned to its stable ESP path
      # (validated at install time below).
      extraEntries = ''
        /Windows
        protocol: efi_chainload
        image_path: boot():/EFI/Microsoft/Boot/bootmgfw.efi
      '';
      # Limine's EFI app checks <app dir>/limine.conf before any drive scan,
      # and on this machine the fallback scan never finds the module's
      # /limine/limine.conf (first boot came up configless). Mirror the
      # freshly written config beside every binary location; this runs in
      # the installer wrapper under `set -e`, so a failed copy fails the
      # install loudly instead of producing an unconfigured boot.
      #
      # And the decisive part: this board boots its "NixOS-boot" option from
      # a CACHED definition -- verified 2026-08-11 by retargeting the NVRAM
      # entry to Limine and watching the firmware load GRUB from the old
      # path anyway. NVRAM is a suggestion box here; the file at the cached
      # path is the only thing it honors. So the active loader's binary is
      # placed AT that path (user-approved). Review-hardened:
      #  - copies come from /boot/EFI/limine/BOOTX64.EFI (the installed,
      #    possibly enrolled/signed binary), not the pristine store file;
      #  - the GRUB backup is created only once, so a limine version bump
      #    cannot clobber it;
      #  - boot-critical writes are tmp+rename and the ESP is syncfs'd;
      #  - /boot/grub/state is removed so flipping the option back forces a
      #    full grub-install, which is what actually restores GRUB at the
      #    cached path (it otherwise skips itself as up-to-date).
      extraInstallCommands = ''
        atomic_cp() {
          ${pkgs.coreutils}/bin/cp "$1" "$2.tmp"
          ${pkgs.coreutils}/bin/mv "$2.tmp" "$2"
        }
        atomic_cp /boot/limine/limine.conf /boot/EFI/limine/limine.conf
        ${pkgs.coreutils}/bin/mkdir -p /boot/EFI/NixOS-boot
        installed=/boot/EFI/limine/BOOTX64.EFI
        target=/boot/EFI/NixOS-boot/GRUBX64.EFI
        if ! ${pkgs.diffutils}/bin/cmp -s "$installed" "$target"; then
          if [ -f "$target" ] && [ ! -f /boot/EFI/NixOS-boot/GRUBX64-BACKUP.EFI ]; then
            atomic_cp "$target" /boot/EFI/NixOS-boot/GRUBX64-BACKUP.EFI
          fi
          atomic_cp "$installed" "$target"
        fi
        atomic_cp /boot/limine/limine.conf /boot/EFI/NixOS-boot/limine.conf
        ${pkgs.coreutils}/bin/rm -f /boot/grub/state
        if ! [ -f /boot/EFI/Microsoft/Boot/bootmgfw.efi ]; then
          echo "limine install: WARNING: /boot/EFI/Microsoft/Boot/bootmgfw.efi missing; the /Windows entry is dead" >&2
        fi
        ${pkgs.coreutils}/bin/sync -f /boot
      '';
    };
  };
}
