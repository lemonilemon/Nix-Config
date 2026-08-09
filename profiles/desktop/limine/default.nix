{ pkgs, lib, ... }:
let
  # Bespoke boot wallpaper: deterministic mocha pixel art (seed 387). The
  # script re-parses its own output's PNG header -- exactly 1920x1080, 8-bit
  # truecolor -- and proves the validator against a known-bad palettized PNG,
  # because bootloader image decoders reject quietly (the GRUB lesson).
  wallpaper = pkgs.runCommand "spacenix-boot-wallpaper" {
    nativeBuildInputs = [ (pkgs.python3.withPackages (ps: [ ps.pillow ])) ];
  } ''python3 ${./wallpaper.py} "$out"'';
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
        branding = "SpaceNix";
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
        margin = 48;
        marginGradient = 24;
      };
    };
    # Limine has no os-prober; Windows is pinned to its stable ESP path
    # (present at /boot/EFI/Microsoft, checked 2026-08-10).
    extraEntries = ''
      /Windows
      protocol: efi_chainload
      image_path: boot():/EFI/Microsoft/Boot/bootmgfw.efi
    '';
  };
}
