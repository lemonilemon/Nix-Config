{ pkgs
, username
, hostname
, lib
, ...
}:
{
  wsl = {
    enable = true;
    defaultUser = username;
    nativeSystemd = true;
  };
  networking.hostName = hostname;
  programs.zsh.enable = true;
  environment.shells = [ pkgs.zsh ];
  environment.systemPackages = with pkgs; [
    wsl-open
  ];
  environment.sessionVariables = {
    NIXHOST = "NixOS-wsl";
  };
  security.sudo.wheelNeedsPassword = false;

  users.users.${username} = {
    isNormalUser = true;
    shell = pkgs.zsh;
    extraGroups = [
      "wheel"
    ];
  };

  time.timeZone = "Asia/Taipei";
  system.stateVersion = "23.05";
}
