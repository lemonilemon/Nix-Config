{
  lib,
  ...
}:
{
  programs.nixvim = {
    plugins = {
      markdown-preview = {
        enable = true;
        settings = {
          auto_close = 1;
          auto_start = 0;
          browser = "firefox";
          filetypes = [ "markdown" ];

          echo_preview_url = 1;
          highlight_css = {
            __raw = "vim.fn.expand('~/highlight.css')";
          };
          markdown_css = "/Users/username/markdown.css";
          page_title = "「\${name}」";
          port = "8080";
          preview_options = {
            disable_filename = 1;
            disable_sync_scroll = 1;
            sync_scroll_type = "middle";
          };
          theme = "dark";
        };

      };
      obsidian = {
        enable = lib.mkDefault false;
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
