{
  lib,
  sys,
  config,
  pkgs,
  ...
}:
{

  options = {
    cli-settings.git.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.cli-settings.enable;
      description = "Enable git settings";
    };
  };
  config = lib.mkIf config.cli-settings.git.enable {
    programs = {
      git = {
        enable = true;
        userName = "lemonilemon";
        userEmail = "imlemonilemon@gmail.com";
        ignores = [
          "*.swp"
          ".*"
        ];
        extraConfig = {
          core = {
            defaultBranch = "main";
            sshCommand = if sys == "wsl" then "ssh.exe" else "ssh";
            pull.rebase = true;
          };
        };
      };
      lazygit.enable = true;
      gh = {
        enable = true;
        extensions = [
          pkgs.gh-copilot
        ];
        settings = {
          git_protocol = "ssh";
        };
      };
    };
  };
}
