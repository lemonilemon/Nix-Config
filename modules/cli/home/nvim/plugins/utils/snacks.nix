{
  lib,
  config,
  ...
}:
{
  programs.nixvim = {
    plugins = {
      snacks = {
        enable = true;
        settings = {
          bigfile = {
            enabled = true;
          };
          quickfile = {
            enabled = true;
          };
          indent = {
            enabled = true;
          };
          input = {
            enabled = true;
          };
          notifier = {
            enabled = true;
          };
          scope = {
            enabled = true;
          };
          scroll = {
            enabled = true;
          };
          statuscolumn = {
            enabled = true;
          };
          words = {
            enabled = true;
          };
          image = lib.mkIf (config.home.cli.multiplexer.program == "tmux") {
            # Image viewer using Kitty Graphics Protocol
            enabled = true; # Not supported by zellij, but tmux is okay
          };
          rename = {
            # file renaming
            enabled = true;
          };
          bufdelete = {
            enabled = true;
          };
        };
      };
    };
  };
}
