{
  lib,
  config,
  pkgs,
  ...
}:
let
  textTypes = [
    "text/plain"
    "text/markdown"
    "text/csv"
    "text/tab-separated-values"
    "text/x-log"
    "text/x-patch"
    "application/json"
    "application/toml"
    "application/yaml"
    "application/xml"
    "text/x-tex"
    "text/x-bibtex"
    "text/vnd.typst"
    "text/x-shellscript"
    "text/x-makefile"
    "text/x-python"
    "text/x-python3"
    "text/x-csrc"
    "text/x-chdr"
    "text/x-c++src"
    "text/x-c++hdr"
    "text/x-go"
    "text/rust"
    "text/x-lua"
    "text/x-nix"
    "text/javascript"
  ];
in
{
  imports = [
    ./options/opts.nix
    ./options/keymaps.nix
    ./options/colorschemes.nix

    ./plugins/lsp/lsp.nix
    ./plugins/lsp/trouble.nix

    ./plugins/formatting/conform.nix

    ./plugins/snippets/luasnip.nix

    ./plugins/completion/cmp.nix
    ./plugins/completion/lspkind.nix

    ./plugins/ai/CopilotChat.nix
    ./plugins/ai/avante.nix

    ./plugins/code/typst.nix
    ./plugins/code/markdown.nix
    ./plugins/code/jupytext.nix
    ./plugins/code/vimtex.nix
    ./plugins/code/competitest.nix
    ./plugins/code/leetcode.nix
    ./plugins/code/coderunner.nix
    ./plugins/code/template.nix

    ./plugins/treesitter/treesitter.nix

    ./plugins/ui/bufferline.nix
    ./plugins/ui/lualine.nix
    ./plugins/ui/noice.nix
    ./plugins/ui/notify.nix

    ./plugins/files/neotree.nix
    ./plugins/files/telescope.nix

    ./plugins/utils/whichkey.nix
    ./plugins/utils/flash.nix
    ./plugins/utils/transparent.nix
    ./plugins/utils/colorizer.nix
    ./plugins/utils/comment.nix
    ./plugins/utils/comment-box.nix
    ./plugins/utils/edgy.nix
    ./plugins/utils/snacks.nix
    ./plugins/utils/zellij-nav.nix
    ./plugins/utils/tmux.nix
    ./plugins/utils/hardtime.nix

    ./plugins/git/lazygit.nix
    ./plugins/git/gitsigns.nix
  ];
  config = lib.mkIf config.home.cli.nvim.enable {
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
      nixpkgs.pkgs = pkgs;

    };

    home.packages = with pkgs; [ neovim-remote ];
    xdg.mimeApps = lib.mkIf config.desktop.enable {
      enable = true;
      defaultApplications = lib.genAttrs textTypes (_: [ "nvim.desktop" ]);
    };
    xdg.desktopEntries = lib.mkIf config.desktop.enable {
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
        mimeType = textTypes;
      };
    };
  };
}
