{
  isWSL,
  ...
}:
{
  programs.atuin = {
    enable = true;
    enableZshIntegration = true;
    daemon.enable = !isWSL;
    # Let fzf own Ctrl-R; Atuin still records history and is reachable via its
    # up-arrow binding. Prevents the fzf/Atuin Ctrl-R conflict warning.
    flags = [ "--disable-ctrl-r" ];
    settings = {
      auto_sync = true;
      sync_frequency = "5m";
      sync_address = "https://api.atuin.sh";
    };
  };
}
