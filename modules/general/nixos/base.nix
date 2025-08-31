{
  inputs,
  lib,
  username,
  pkgs,
  hostname,
  ...
}:
{
  nix.registry.nixpkgs.flake = inputs.nixpkgs;
  nix.channel.enable = false;
  # Basic hardware and system settings
  hardware.graphics.enable32Bit = lib.mkDefault true;
  services.dbus.implementation = lib.mkDefault "broker";
  networking.hostName = lib.mkDefault hostname;

  # Shell configuration
  programs.zsh.enable = lib.mkDefault true;
  environment.shells = [ pkgs.zsh ];

  # Security settings
  security.sudo.wheelNeedsPassword = lib.mkDefault false;

  # Container support
  virtualisation.containers.enable = lib.mkDefault true;
  virtualisation = {
    podman = {
      enable = lib.mkDefault true;
      # Create a `docker` alias for podman, to use it as a drop-in replacement
      # dockerCompat = true;
      # Required for containers under podman-compose to be able to talk to each other.
      defaultNetwork.settings.dns_enabled = lib.mkDefault true;
    };
  };

  # User configuration
  users.users.${username} = {
    isNormalUser = true;
    shell = pkgs.zsh;
    extraGroups = [
      "wheel"
      "docker"
      "networkmanager"
    ];
    group = username;
  };

  # Create user group
  users.groups.${username} = { };

  # Default time zone (can be overridden by hosts)
  time.timeZone = lib.mkDefault "Asia/Taipei";

  # System state version
  system.stateVersion = "25.11";
}
