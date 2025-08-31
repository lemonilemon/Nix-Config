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
        userName = "lemonilemon";
        userEmail = "imlemonilemon@gmail.com";
        ignores = [
          "*.swp"
          ".*"
        ];
        extraConfig = {
          init = {
            defaultBranch = "main";
          };
          core = {
            sshCommand = if isWSL then "ssh.exe" else "ssh";
            editor = "nvim";
          };
          pull.rebase = true;
        };
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
