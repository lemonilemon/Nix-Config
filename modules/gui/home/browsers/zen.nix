{
  config,
  lib,
  inputs,
  system,
  ...
}:
{
  config = lib.mkIf config.home.gui.browsers.zen.enable {
    home.packages = [
      inputs.zen-browser.packages."${system}".default
    ];
  };
}
