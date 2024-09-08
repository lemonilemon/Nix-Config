{
    programs.nixvim.plugins = {
        # https://nix-community.github.io/nixvim/plugins/nvim-tree/index.html
        nvim-tree = {
            enable = true;
            autoClose = true;
            autoReloadOnWrite = true;
            sortBy = "case_sensitive";
            filters = {
                dotfiles = false;
            };
            renderer = {
                groupEmpty = true;
            };
            view = {
                width = 40;
            };
        };

    };
}
