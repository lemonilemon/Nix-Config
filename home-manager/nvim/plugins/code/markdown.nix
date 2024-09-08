{
  programs.nixvim = {
    plugins = {
      markdown-preview = {
        enable = true;
        settings = {
          auto_close = true;
          auto_start = true;
          browser = "firefox";
          filetypes = [ "markdown" ];

          echo_preview_url = true;
          highlight_css = {
            __raw = "vim.fn.expand('~/highlight.css')";
          };
          markdown_css = "/Users/username/markdown.css";
          page_title = "「\${name}」";
          port = "8080";
          preview_options = {
            disable_filename = true;
            disable_sync_scroll = true;
            sync_scroll_type = "middle";
          };
          theme = "dark";
        };

      };
      obsidian = {
        enable = true;
        settings = {
          completion = {
            min_chars = 2;
            nvim_cmp = true;
          };
          new_notes_location = "current_dir";
          workspaces = [
            {
              name = "school";
              path = "~/obsidian/school";
            }
          ];
        };
      };
    };
  };
}
