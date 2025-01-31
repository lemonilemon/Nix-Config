{
  config,
  pkgs,
  lib,
  ...
}:
{
  imports = [
    ./git
    ./nvim
    ./programs
    ./shells
    ./zellij
  ];
  options = {
    cli-settings.enable = lib.mkOption {
      type = lib.types.bool;
      default = lib.mkDefault true;
      description = "Enable my CLI settings";
    };
  };
  config = lib.mkIf config.cli-settings.enable {
    # All of the cli settings are default to be enabled
    # cli-settings = {
    #   git.enable = lib.mkDefault true;
    #   nvim.enable = lib.mkDefault true;
    #   programs.enable = lib.mkDefault true;
    #   shells.enable = lib.mkDefault true;
    #   zellij.enable = lib.mkDefault true;
    # };
  };

}
