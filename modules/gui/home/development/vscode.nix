{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.gui.development.vscode.enable {
    programs.vscode = {
      enable = true;
      package = pkgs.vscode;
    };
  };
}
