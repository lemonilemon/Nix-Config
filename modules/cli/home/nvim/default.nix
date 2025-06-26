{
  lib,
  config,
  pkgs,
  ...
}:
{
  imports = [
    ./options/opts.nix
    ./options/keymaps.nix
    ./options/colorschemes.nix

    # lsp
    ./plugins/lsp/lsp.nix
    ./plugins/lsp/trouble.nix

    # formatting
    ./plugins/formatting/conform.nix

    # snippets
    ./plugins/snippets/luasnip.nix

    # completion
    ./plugins/completion/cmp.nix
    ./plugins/completion/copilot.nix
    ./plugins/completion/lspkind.nix

    # code
    ./plugins/code/typst.nix
    ./plugins/code/markdown.nix
    ./plugins/code/vimtex.nix
    ./plugins/code/competitest.nix
    ./plugins/code/coderunner.nix
    ./plugins/code/template.nix

    # treesitter
    ./plugins/treesitter/treesitter.nix

    # ui
    ./plugins/ui/bufferline.nix
    ./plugins/ui/lualine.nix
    ./plugins/ui/noice.nix
    ./plugins/ui/notify.nix

    # file explorer
    ./plugins/files/neotree.nix
    ./plugins/files/telescope.nix

    # utils
    ./plugins/utils/whichkey.nix
    ./plugins/utils/flash.nix
    ./plugins/utils/transparent.nix
    ./plugins/utils/colorizer.nix
    ./plugins/utils/comment.nix
    ./plugins/utils/comment-box.nix
    ./plugins/utils/edgy.nix
    ./plugins/utils/snacks.nix
    ./plugins/utils/zellij-nav.nix
    ./plugins/utils/hardtime.nix

    # git
    ./plugins/git/lazygit.nix
    ./plugins/git/gitsigns.nix
  ];
  config = lib.mkIf config.home.cli-settings.nvim.enable {
    programs.nixvim = {
      enable = true;
      defaultEditor = true;
      viAlias = true;
      vimAlias = true;
      vimdiffAlias = true;
      # performance tweaks (experimental)
      performance = {
        byteCompileLua.enable = true;
        # combinePlugins.enable = true; # It causes some collisions
      };

    };

    home.packages = with pkgs; [ neovim-remote ];
    xdg.mimeApps = {
      enable = true;
      defaultApplications = {
        "text/plain" = "nvim.desktop";
      };
    };
    xdg.desktopEntries = {
      nvim = {
        name = "Neovim";
        genericName = "Text Editor";
        exec = "kitty nvim %F";
        terminal = false;
        categories = [
          "Utility"
          "TextEditor"
        ];
        icon = "nvim";
        mimeType = [ "text/plain" ];
      };
    };
  };
}
