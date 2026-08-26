{ config, lib, ... }:
let
  cfg = config.nixos.desktop;
in
{
  # The fallback branch: stock GRUB with the vinceliuice theme from
  # profiles/boot.nix. The bespoke SpaceNix GRUB theme was dropped on
  # 2026-08-11 after Limine won the trial (recoverable from git history).
  # Flipping the option here still works as the escape hatch: the limine
  # branch removes /boot/grub/state on install, so this branch's switch
  # forces a full grub-install that restores GRUB at the firmware's
  # cached boot path.
  config = lib.mkIf (cfg.bootloader == "grub") {
    boot.loader.limine.enable = false;
    # Sizes the generated theme assets. profiles/boot.nix leaves this at
    # mkDefault; the host's real panel mode wins here.
    boot.loader.grub2-theme.customResolution = cfg.bootloaderResolution;
    boot.loader.grub = {
      # Native mode, and hand the framebuffer to the kernel unchanged so
      # the menu -> Plymouth transition has one less modeset flash.
      gfxmodeEfi = "${cfg.bootloaderResolution},auto";
      gfxpayloadEfi = "keep";
    };
  };
}
