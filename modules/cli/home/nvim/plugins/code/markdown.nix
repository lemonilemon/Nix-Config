{
  config,
  ...
}:
let
  vault = config.home.general.obsidian;
in
{
  programs.nixvim = {
    plugins = {
      markdown-preview = {
        enable = true;
        settings = {
          auto_close = 1;
          auto_start = 0;
          browser = "";
          filetypes = [ "markdown" ];

          echo_preview_url = 1;
          highlight_css = {
            __raw = "vim.fn.expand('~/highlight.css')";
          };
          page_title = "「\${name}」";
          preview_options = {
            disable_filename = 1;
            disable_sync_scroll = 1;
            sync_scroll_type = "middle";
          };
          theme = "dark";
        };

      };
      obsidian = {
        # Follows the vault flag rather than being independently toggled: the
        # plugin is only useful where the vault actually exists, and the path
        # below comes from the same option Syncthing uses, so the editor and
        # the sync folder cannot end up pointed at different directories.
        enable = vault.enable;
        settings = {
          completion = {
            min_chars = 2;
          };
          new_notes_location = "current_dir";
          workspaces = [
            {
              name = "notes";
              path = vault.vaultPath;
            }
          ];
        };
      };
    };
  };
}
