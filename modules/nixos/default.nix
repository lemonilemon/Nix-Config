{
  lib,
  ...
}:
{
  imports = [
    ./gui
    ./general
  ];
  config = {
    catppuccin.enable = lib.mkDefault true;
    catppuccin.flavor = lib.mkDefault "mocha";
    gui-settings.enable = true;
    virtualisation.docker = {
      enable = true;
      enableOnBoot = true;
      autoPrune.enable = true;
    };
  };
}
