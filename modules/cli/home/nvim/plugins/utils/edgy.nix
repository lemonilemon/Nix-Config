{
  config,
  lib,
  ...
}:
let
  helpers = config.lib.nixvim;
in
{
  programs.nixvim = {
    plugins.edgy = {
      enable = lib.mkDefault true;
      settings = {
        bottom = [
          {
            ft = "toggleterm";
            filter = helpers.mkRaw ''
              function(buf, win)
                return vim.api.nvim_win_get_config(win).relative == ""
              end
            '';
          }
        ];
        left = [
          {
            title = "Neo-Tree";
            ft = "neo-tree";
            filter = helpers.mkRaw ''
              function(buf)
                return vim.b[buf].neo_tree_source == "filesystem"
              end
            '';
          }
          {
            title = "Neo-Tree Git";
            ft = "neo-tree";
            filter = helpers.mkRaw ''
              function(buf)
                return vim.b[buf].neo_tree_source == "git_status"
              end
            '';
            open = "Neotree position=right git_status";
          }
          {
            title = "Neo-Tree Buffers";
            ft = "neo-tree";
            filter = helpers.mkRaw ''
              function(buf)
                return vim.b[buf].neo_tree_source == "buffers"
              end
            '';
            open = "Neotree position=top buffers";
          }
          "neo-tree"
        ];
        options = {
          left = {
            size = 30;
          };
          right = {
            size = 30;
          };
        };
      };
    };
  };
}
