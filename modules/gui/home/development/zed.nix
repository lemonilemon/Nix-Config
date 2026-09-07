{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.gui.development.zed.enable {
    programs.zed-editor = {
      enable = true;
      package = pkgs.zed-editor;
      # defaultEditor stays off: nvim owns EDITOR/VISUAL (modules/cli/home/nvim).
    };
  };
}
