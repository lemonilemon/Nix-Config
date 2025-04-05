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
      enableDiagnostics = true;
      enableGitStatus = true;
      enableModifiedMarkers = true;
      enableRefreshOnWrite = true;
      closeIfLastWindow = true;
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
