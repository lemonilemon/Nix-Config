{
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
    ./plugins/code/markdown.nix
    ./plugins/code/vimtex.nix
    ./plugins/code/competitest.nix
    ./plugins/code/coderunner.nix
    ./plugins/code/template.nix

    # treesitter
    ./plugins/treesitter/treesitter.nix

    # ui
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

    # git
    ./plugins/git/lazygit.nix
    ./plugins/git/gitsigns.nix
  ];
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
  home.packages = with pkgs; [
    neovim-remote
  ];
}
