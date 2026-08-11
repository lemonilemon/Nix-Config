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

    # Deliberately NOT catppuccin.limine: at the pinned revs its theme path
    # does not exist (the port moved its confs into themes/), and its opaque
    # term_background would defeat the translucent panel below. The style IS
    # catppuccin-mocha, applied directly through the typed options.
    catppuccin.limine.enable = lib.mkForce false;

    boot.loader.limine = {
      enable = true;
      maxGenerations = 5; # matches the GRUB configurationLimit
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
          background = "C71E1E2E"; # translucent panel over the wallpaper
          # The menu floats as a card under the wordmark; entries render in
          # Spleen doubled to 16x32 on screen.
          margin = 320;
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
      # (present at /boot/EFI/Microsoft, checked 2026-08-10).
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
      # placed AT that path (user-approved). GRUB's real binary survives as
      # GRUBX64-BACKUP.EFI (bootable via the firmware's file browser), and
      # flipping the option back restores it because grub-install rewrites
      # its own path.
      extraInstallCommands = ''
        ${pkgs.coreutils}/bin/cp /boot/limine/limine.conf /boot/EFI/limine/limine.conf
        ${pkgs.coreutils}/bin/mkdir -p /boot/EFI/NixOS-boot
        if [ -f /boot/EFI/NixOS-boot/GRUBX64.EFI ] \
          && ! ${pkgs.diffutils}/bin/cmp -s ${pkgs.limine}/share/limine/BOOTX64.EFI /boot/EFI/NixOS-boot/GRUBX64.EFI; then
          ${pkgs.coreutils}/bin/cp /boot/EFI/NixOS-boot/GRUBX64.EFI /boot/EFI/NixOS-boot/GRUBX64-BACKUP.EFI
        fi
        ${pkgs.coreutils}/bin/cp ${pkgs.limine}/share/limine/BOOTX64.EFI /boot/EFI/NixOS-boot/GRUBX64.EFI
        ${pkgs.coreutils}/bin/cp /boot/limine/limine.conf /boot/EFI/NixOS-boot/limine.conf
      '';
    };
  };
}
