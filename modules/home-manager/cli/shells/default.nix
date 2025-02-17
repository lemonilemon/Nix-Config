{
  lib,
  config,
  ...
}:
{
  imports = [
    ./zsh.nix
    ./starship.nix
    ./zoxide.nix
  ];
  options = {
    cli-settings.shells.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.cli-settings.enable;
      description = "Enable shell configuration";
    };
  };
  config = lib.mkIf config.cli-settings.shells.enable { };
}
