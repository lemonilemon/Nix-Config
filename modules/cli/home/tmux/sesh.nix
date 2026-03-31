{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.cli.multiplexer.tmux.enable {
    programs.sesh = {
      enable = true;
      enableTmuxIntegration = true;
      icons = true;
      tmuxKey = "T";
    };
  };
}
