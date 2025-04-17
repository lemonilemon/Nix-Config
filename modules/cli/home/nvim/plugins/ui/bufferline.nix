{ config, ... }:
let
  helpers = config.lib.nixvim;
in
{
  programs.nixvim = {
    plugins = {
      bufferline = {
        enable = true;
        settings = {
          options = {
            close_command = helpers.mkRaw ''
              function(opts) 
                  Snacks.bufdelete.delete(opts) 
              end
            '';
            right_mouse_command = helpers.mkRaw ''
              function(opts) 
                  Snacks.bufdelete.delete(opts) 
              end
            '';
            diagnostics = "nvim_lsp";
            always_show_bufferline = false;
            offsets = [
              {
                filetype = "neo-tree";
                text = "Neo-tree";
                highlight = "Directory";
                text_align = "left";
              }
              {
                filetype = "snacks_layout_box";
              }
            ];
          };
        };

      };
    };
  };
}
