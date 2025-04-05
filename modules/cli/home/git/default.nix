{
  lib,
  sys,
  config,
  pkgs,
  ...
}:
{

  config = lib.mkIf config.home.cli-settings.git.enable {
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
            sshCommand = if sys == "NixOS-wsl" then "ssh.exe" else "ssh";
            editor = "nvim";
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
