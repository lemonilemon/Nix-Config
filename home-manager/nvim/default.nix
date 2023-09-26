{
    imports = [
        ./lsp
    ];
    home.file.".config/nvim" = {
        source = ./source;
        recursive = true;
    };
    programs.neovim = {
        enable = true;
        viAlias = true;
        vimAlias = true;
        vimdiffAlias = true;
        defaultEditor = true;
    };
}
