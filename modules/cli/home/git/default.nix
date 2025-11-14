{
  lib,
  config,
  pkgs,
  isWSL,
  ...
}:
{

  config = lib.mkIf config.home.cli-settings.git.enable {
    programs = {
      git = {
        enable = true;
        settings = {
          user = {
            name = "lemonilemon";
            email = "imlemonilemon@gmail.com";
          };
          init = {
            defaultBranch = "main";
          };
          core = {
            sshCommand = if isWSL then "ssh.exe" else "ssh";
            editor = "nvim";
            autocrlf = true;
          };
          pull.rebase = true;
        };
        ignores = [
          "*.swp"
          ".*"
        ];
      };
      git.lfs.enable = true; # Git LFS (Large File Storage)
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
