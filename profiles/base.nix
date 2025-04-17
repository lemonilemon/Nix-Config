{
  lib,
  username,
  pkgs,
  hostname,
  ...
}:
{
  # Allow unfree and broken packages system wide
  nixpkgs.config = {
    allowUnfree = true;
    allowBroken = true;
  };
  hardware.graphics.enable32Bit = lib.mkDefault true;
  services.dbus.implementation = lib.mkDefault "broker";
  networking.hostName = lib.mkDefault hostname;
  programs.zsh.enable = lib.mkDefault true;
  environment.shells = [ pkgs.zsh ];
  security.sudo.wheelNeedsPassword = lib.mkDefault false;

  virtualisation.containers.enable = true;
  virtualisation = {
    podman = {
      enable = true;

      # Create a `docker` alias for podman, to use it as a drop-in replacement
      # dockerCompat = true;

      # Required for containers under podman-compose to be able to talk to each other.
      defaultNetwork.settings.dns_enabled = true;
    };
  };

  users.users.${username} = {
    isNormalUser = true;
    shell = pkgs.zsh;
    extraGroups = [
      "wheel"
      "docker"
      "networkmanager"
    ];
  };

  time.timeZone = "Asia/Taipei";
  system.stateVersion = "25.05";
}
