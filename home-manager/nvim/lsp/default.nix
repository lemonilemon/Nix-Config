{ pkgs, ... }:
{
    programs.neovim.extraPackages = with pkgs; [
        # Language servers
        lua-language-server # lua
        pyright # python
        ccls # c/cpp
        nodePackages.bash-language-server # bash
        cmake-language-server # cmake
        marksman # markdown
        ltex-ls # latex
        nil # nix
        nodePackages.vim-language-server # vim
        rust-analyzer # rust
        nodePackages.typescript-language-server # typescript
    ];
}
