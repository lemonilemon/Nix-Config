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
  environment.sessionVariables = {
    NIXHOST = "NixOS-wsl";
  };
  security.sudo.wheelNeedsPassword = lib.mkDefault false;

  users.users.${username} = {
    isNormalUser = true;
    shell = pkgs.zsh;
    extraGroups = [
      "wheel"
      "docker"
    ];
  };

  time.timeZone = "Asia/Taipei";
  system.stateVersion = "25.05";
}
