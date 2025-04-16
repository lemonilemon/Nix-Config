{
  programs.nixvim = {
    plugins = {
      bufferline = {
        enable = true;
        settings = {
          options = {
            close_command = ''function(n) Snacks.bufdelete(n) end'';
            right_mouse_command = ''function(n) Snacks.bufdelete(n) end'';
            diagnostics = "nvim_lsp";
            always_show_bufferline = true;
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
