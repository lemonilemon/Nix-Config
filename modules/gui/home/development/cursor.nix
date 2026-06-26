{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.gui.development.cursor.enable {
    programs.vscode = {
      enable = true;
      package = pkgs.code-cursor;
    };
  };
}
