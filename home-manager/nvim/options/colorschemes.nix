{
    programs.nixvim = {
        colorschemes = {
            # https://nix-community.github.io/nixvim/colorschemes/tokyonight/index.html
            tokyonight = {
                enable = false;
                settings = {
                    style = "night";
                    transparent = true;
                    terminal_colors = true;
                };
            };
            catppuccin = {
                enable = true;
                settings = {
                    flavour = "mocha";
                    integrations = {
                        cmp = true;
                        gitsigns = true;
                        mini = {
                            enabled = true;
                            indentscope_color = "";
                        };
                        notify = true;
                        nvimtree = true;
                        treesitter = true;
                    };
                    transparent_background = true;
                    term_colors = true;
                };
            };
        };
    };
}
