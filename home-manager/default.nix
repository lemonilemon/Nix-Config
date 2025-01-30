{
  username,
  ...
}:
{
  home.username = "${username}";
  home.homeDirectory = "/home/${username}";
  imports = [
    ./cli/nvim
    ./cli/shells
    ./cli/git
    ./cli/programs
    ./cli/zellij
    ./general
  ];
  home.stateVersion = "24.11";
  programs.home-manager.enable = true;

  nixpkgs.config.allowUnfree = true;
  nixpkgs.config.allowUnfreePredicate = _: true;

  general-settings.enable = true;
}
