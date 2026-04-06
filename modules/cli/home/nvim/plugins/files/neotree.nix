{
  config,
  lib,
  ...
}:
{
  programs.nixvim.plugins = {
    web-devicons.enable = true;
    mini = {
      enable = true;
      mockDevIcons = true;
      modules.icons = {
        style = "glyph";
      };
    };
    # https://nix-community.github.io/nixvim/plugins/nvim-tree/index.html
    neo-tree = {
      enable = lib.mkDefault true;
      settings = {
        enable_diagnostics = true;
        enable_git_status = true;
        enable_modified_markers = true;
        enable_refresh_on_write = true;
        close_if_last_window = true;
      };
    };
    edgy.settings = {
      left = [
        {
          title = "Neo-Tree";
          ft = "neo-tree";
        }
      ];
    };
  };
}
