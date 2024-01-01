{
    pkgs,
    ...
}: {
    home.packages = with pkgs; [
        neovim-remote
    ];
    imports = [
        ./lsp
    ];
    home.file.".config/nvim" = {
    	enable = true;
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
