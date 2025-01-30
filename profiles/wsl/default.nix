{
  pkgs,
  username,
  hostname,
  ...
}:
{
  wsl = {
    enable = true;
    defaultUser = username;
    wslConf.automount.root = "/mnt";
    wslConf.interop.appendWindowsPath = true;
    wslConf.network.generateHosts = false;
    startMenuLaunchers = true;
  };
  services.dbus.implementation = "broker";
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
  system.stateVersion = "25.05";
}
