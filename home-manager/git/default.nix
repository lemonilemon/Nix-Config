{
  WSL,
  pkgs,
  ...
}:
{
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
          sshCommand = if WSL then "ssh.exe" else "ssh";
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
}
