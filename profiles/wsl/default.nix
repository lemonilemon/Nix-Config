{
  pkgs,
  username,
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
    docker-desktop.enable = true;
  };
  environment.systemPackages = with pkgs; [
    wsl-open
  ];
  environment.sessionVariables = {
    NIXHOST = "NixOS-wsl";
  };
  # cli-settings.enable = true;
  # general-settings.enable = true;
  # gui-settings.enable = false;
  # gui-settings.browsers.enable = true;
}
