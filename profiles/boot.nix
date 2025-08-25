{
  lib,
  pkgs,
  ...
}:
{
  # bootloader
  boot = {
    plymouth = {
      enable = true;
      theme = "connect";
      themePackages = with pkgs; [
        # By default we would install all themes
        (adi1090x-plymouth-themes.override {
          selected_themes = [ "connect" ];
        })
      ];
    };
    kernelPackages = pkgs.linuxPackages_latest;
    kernelParams = [
      # Not to suspend usb devices
      "usbcore.autosuspend=-1"
      "usbcore.autosuspend_delay_ms=-1"
      # For boot
      "quiet"
      "splash"
      "loglevel=3"
      "boot.shell_on_fail"
      "udev.log_priority=3"
      "rd.systemd.show_status=auto"
    ];
    consoleLogLevel = 3;
    initrd.verbose = false;
    # Reduce initrd size and improve loading speed
    initrd.compressor = "zstd";
    initrd.compressorArgs = [
      "-19"
      "-T0"
    ];
    initrd.systemd.enable = true;
    loader = {
      timeout = 10; # Timeout for the bootloader menu
      efi = {
        canTouchEfiVariables = true; # Allow writing to EFI variables
      };
      systemd-boot = {
        enable = lib.mkDefault false; # Don't use systemd-boot by default
        configurationLimit = 5;
      };
      grub = {
        enable = lib.mkDefault true;
        useOSProber = lib.mkDefault true;
        efiSupport = lib.mkDefault true;
        device = lib.mkDefault "nodev";
        configurationLimit = lib.mkDefault 5;
      };
      grub2-theme = {
        enable = lib.mkDefault true;
        theme = lib.mkDefault "stylish";
        footer = lib.mkDefault false;
        customResolution = lib.mkDefault "1920x1080";
      };
    };
  };
  catppuccin.plymouth.enable = lib.mkDefault false; # use plymouth theme instead
  catppuccin.grub.enable = lib.mkDefault false; # use grub2-theme instead
}
