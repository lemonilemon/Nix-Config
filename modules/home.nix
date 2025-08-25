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
    ./cli/home
    ./gui/home
    ./desktop/home
    ./general/home
  ];
  config = lib.mkIf config.home.enable {
    home.username = lib.mkDefault "${username}";
    home.homeDirectory = lib.mkDefault "/home/${username}";
    home.stateVersion = lib.mkDefault "25.05";
    programs.home-manager.enable = true;
    catppuccin = {
      enable = true;
      flavor = "mocha";
    };
  };
}
