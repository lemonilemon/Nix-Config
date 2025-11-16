{
  lib,
  username,
  config,
  catppuccin,
  ...
}:
{
  # Mainly for home-manager, which imports all the home-manager modules
  imports = [
    ./options.nix
    ./cli/home
    ./gui/home
    ./desktop/home
    ./general/home
  ];
  config = lib.mkIf config.home.enable {
    home.username = "${username}";
    home.homeDirectory = "/home/${username}";
    home.stateVersion = "25.11";
    programs.home-manager.enable = true;
    catppuccin = {
      enable = true;
      flavor = "mocha";
    };
  };
}
