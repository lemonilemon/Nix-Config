{ config, lib, ... }:
{
  # The fallback branch: stock GRUB with the vinceliuice theme from
  # profiles/boot.nix. The bespoke SpaceNix GRUB theme was dropped on
  # 2026-08-11 after Limine won the trial (recoverable from git history).
  # Flipping the option here still works as the escape hatch: the limine
  # branch removes /boot/grub/state on install, so this branch's switch
  # forces a full grub-install that restores GRUB at the firmware's
  # cached boot path.
  config = lib.mkIf (config.nixos.desktop.bootloader == "grub") {
    boot.loader.limine.enable = false;
    boot.loader.grub = {
      # Native mode, and hand the framebuffer to the kernel unchanged so
      # the menu -> Plymouth transition has one less modeset flash.
      gfxmodeEfi = "1920x1080,auto";
      gfxpayloadEfi = "keep";
    };
  };
}
