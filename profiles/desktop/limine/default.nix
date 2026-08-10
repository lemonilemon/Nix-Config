{ pkgs, lib, ... }:
let
  # Bespoke boot wallpaper: deterministic mocha pixel art (seed 387) plus the
  # antialiased SpaceNix wordmark -- Limine cannot render TTFs, so the
  # typography lives in the image. The script re-parses its own output's PNG
  # header -- exactly 1920x1080, 8-bit truecolor -- and proves the validator
  # against a known-bad palettized PNG, because bootloader image decoders
  # reject quietly (the GRUB lesson).
  wallpaper =
    pkgs.runCommand "spacenix-boot-wallpaper"
      {
        nativeBuildInputs = [ (pkgs.python3.withPackages (ps: [ ps.pillow ])) ];
      }
      ''python3 ${./wallpaper.py} "$out" ${pkgs.jetbrains-mono}/share/fonts/truetype/JetBrainsMono-SemiBold.ttf'';

  # Menu font: Spleen 8x16 converted to Limine's raw CP437 format, with the
  # required menu glyphs verified present (Tamsyn, for one, lacks the arrows).
  terminalFont = pkgs.runCommand "limine-term-font-spleen" {
    nativeBuildInputs = [ pkgs.python3 ];
  } ''python3 ${./font.py} ${pkgs.spleen}/share/consolefonts/spleen-8x16.psfu "$out"'';
in
{
  # Limine trial (2026-08): GRUB's files stay on the ESP, so the firmware
  # boot menu can still reach it; flipping these two options back and
  # rebuilding restores it outright.
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
        # The wordmark in the wallpaper is the branding; this suppresses
        # Limine's own line. To verify at first boot: if an empty string
        # still renders the "Limine 12.5.2" default, accept the small line.
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
  };
}
