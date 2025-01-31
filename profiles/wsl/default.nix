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
  };
  environment.systemPackages = with pkgs; [
    wsl-open
  ];
}
