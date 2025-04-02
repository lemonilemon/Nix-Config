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
    home-manager.backupFileExtension = "backup";

    catppuccin.enable = lib.mkDefault true;
    catppuccin.flavor = lib.mkDefault "mocha";
    virtualisation.docker = {
      enable = true;
      enableOnBoot = true;
      autoPrune.enable = true;
    };
  };
}
