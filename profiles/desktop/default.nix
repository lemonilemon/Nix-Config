# Edit this configuration file to define what should be installed on
# your system. Help is available in the configuration.nix(5) man page, on
# https://search.nixos.org/options and in the NixOS manual (`nixos-help`).

{
  pkgs,
  config,
  hostname,
  lib,
  ...
}:
{
  imports = [
    # Include the results of the hardware scan.
    ./hardware-configuration.nix
    ./config.nix
    ./nvidia.nix # For NVIDIA graphics cards
    ../base.nix
    ../i18n.nix
  ];

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
        # configurationLimit = 5;
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

  networking.wireless.enable = false; # explicitly disable wireless
  networking.networkmanager.enable = true; # Easiest to use and most distros use this by default.

  environment.sessionVariables = {
    NIXHOST = "desktop";
  };

  # Set your time zone.
  time.timeZone = "Asia/Taipei";

  environment.systemPackages = with pkgs; [
    ntfs3g # NTFS driver
    pciutils # Inspecting PCI devices
    mesa # 3D graphics library
    glxinfo # Test utilities for OpenGL
  ];

  # Power
  powerManagement = {
    enable = true;
    powertop.enable = true;
    cpuFreqGovernor = "performance";
  };

  services = {
    thermald.enable = true;
    power-profiles-daemon.enable = false;
    auto-cpufreq = {
      enable = true;
      settings = {
        charger = {
          governor = "performance";
          turbo = "auto";
        };
      };
    };
  };

  # Hardware

  hardware = {
    graphics.enable = true;
    bluetooth.enable = true;
  };
  # Bluetooth
  services.blueman.enable = true;

  # Sound
  services.pulseaudio.enable = false; # Use pipewire instead
  security.rtkit.enable = true;
  services.pipewire = {
    enable = true;
    alsa.enable = true;
    alsa.support32Bit = true;
    pulse.enable = true;
    jack.enable = lib.mkDefault true;
  };

  security.polkit.enable = true;

  # Some programs need SUID wrappers, can be configured further or are
  # started in user sessions.
  # programs.mtr.enable = true;
  # programs.gnupg.agent = {
  #   enable = true;
  #   enableSSHSupport = true;
  # };

  # List services that you want to enable:

  # Enable the OpenSSH daemon.
  # services.openssh.enable = true;

  # Open ports in the firewall.
  # networking.firewall.allowedTCPPorts = [ ... ];
  # networking.firewall.allowedUDPPorts = [ ... ];
  # Or disable the firewall altogether.
  # networking.firewall.enable = false;

  # Copy the NixOS configuration file and link it from the resulting system
  # (/run/current-system/configuration.nix). This is useful in case you
  # accidentally delete configuration.nix.
  # system.copySystemConfiguration = true;

  # This option defines the first version of NixOS you have installed on this particular machine,
  # and is used to maintain compatibility with application data (e.g. databases) created on older NixOS versions.
  #
  # Most users should NEVER change this value after the initial install, for any reason,
  # even if you've upgraded your system to a new NixOS release.
  #
  # This value does NOT affect the Nixpkgs version your packages and OS are pulled from,
  # so changing it will NOT upgrade your system - see https://nixos.org/manual/nixos/stable/#sec-upgrading for how
  # to actually do that.
  #
  # This value being lower than the current NixOS release does NOT mean your system is
  # out of date, out of support, or vulnerable.
  #
  # Do NOT change this value unless you have manually inspected all the changes it would make to your configuration,
  # and migrated your data accordingly.
  #
  # For more information, see `man configuration.nix` or https://nixos.org/manual/nixos/stable/options#opt-system.stateVersion .
  system.stateVersion = "25.05"; # Did you read the comment?

}
