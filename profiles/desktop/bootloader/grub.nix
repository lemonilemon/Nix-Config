{
  pkgs,
  config,
  lib,
  ...
}:
let
  jbm = "${pkgs.jetbrains-mono}/share/fonts/truetype";

  # Same art as the Limine branch: deterministic mocha pixel art plus the
  # antialiased SpaceNix wordmark, validated byte-level (see ../boot-theme/).
  # The expression is identical in both branches, so it is one store path.
  wallpaper = pkgs.runCommand "spacenix-boot-wallpaper" {
    nativeBuildInputs = [ (pkgs.python3.withPackages (ps: [ ps.pillow ])) ];
  } ''python3 ${../boot-theme/wallpaper.py} "$out" ${jbm}/JetBrainsMono-SemiBold.ttf'';

  # The full GRUB theme: JetBrains Mono pf2 fonts (1-bit, but real letterforms
  # at menu size), translucent panel and selection 9-slices, per-class icons,
  # and a theme.txt generated FROM the pf2 NAME sections -- the two silent
  # failure modes of the July attempt are both build failures here.
  theme =
    pkgs.runCommand "spacenix-grub-theme"
      {
        nativeBuildInputs = [
          pkgs.grub2
          (pkgs.python3.withPackages (ps: [ ps.pillow ]))
        ];
      }
      ''
        mkdir -p "$out"
        grub-mkfont --size 30 --output "$out/jbm-item.pf2" ${jbm}/JetBrainsMono-Medium.ttf
        grub-mkfont --size 17 --output "$out/jbm-small.pf2" ${jbm}/JetBrainsMono-Regular.ttf
        python3 ${./theme.py} "$out" ${wallpaper} "$out/jbm-item.pf2" "$out/jbm-small.pf2"
      '';
in
{
  config = lib.mkIf (config.nixos.desktop.bootloader == "grub") {
    boot.loader.limine.enable = false;
    boot.loader.grub2-theme.enable = false; # replaced by the in-repo theme
    boot.loader.grub = {
      theme = theme;
      # Native mode, and hand the framebuffer to the kernel unchanged so the
      # menu -> Plymouth transition has one less modeset flash.
      gfxmodeEfi = "1920x1080,auto";
      gfxpayloadEfi = "keep";
    };
  };
}
