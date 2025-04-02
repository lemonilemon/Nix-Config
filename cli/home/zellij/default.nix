{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.cli-settings.zellij.enable {
    programs.zellij = {
      enable = true;
      enableZshIntegration = true;
      attachExistingSession = true;
      exitShellOnExit = true;
      settings = {
        show_startup_tips = false;
        theme = "catppuccin-mocha";
        default_shell = "zsh";
      };
    };
  };
}
