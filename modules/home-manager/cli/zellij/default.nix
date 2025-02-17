{
  lib,
  config,
  ...
}:
{
  options = {
    cli-settings.zellij.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.cli-settings.enable;
      description = "Enable zellij settings";
    };
  };
  config = lib.mkIf config.cli-settings.zellij.enable {
    programs.zellij = {
      enable = true;
      settings = {
        theme = "catppuccin-mocha";
        default_shell = "zsh";
      };
    };
  };
}
