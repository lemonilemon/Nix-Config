{
  isWSL,
  ...
}:
{
  programs.atuin = {
    enable = true;
    enableZshIntegration = true;
    daemon.enable = !isWSL;
    settings = {
      auto_sync = true;
      sync_frequency = "5m";
      sync_address = "https://api.atuin.sh";
    };
  };
}
