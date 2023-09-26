{
  pkgs,
  username,
  ...
}: {
  wsl = {
    enable = true;
    defaultUser = "${username}";
  };

  networking.hostName = "wsl";

  environment.systemPackages = with pkgs; [
      wsl-open
  ];

  time.timeZone = "Asia/Taiwan";
  system.stateVersion = "23.05";
}
