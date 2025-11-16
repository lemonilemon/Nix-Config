{
  lib,
  ...
}:
{
  imports = [
    ./cli
    ./gui
    ./general
    ./desktop
    ./options.nix
  ];
  config = {
    catppuccin.enable = lib.mkDefault true;
    catppuccin.flavor = lib.mkDefault "mocha";
  };
}
