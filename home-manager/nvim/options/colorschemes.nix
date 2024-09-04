{
    programs.nixvim = {
        colorschemes = {
            tokyonight = {
                enable = true;
                settings = {
                    # https://nix-community.github.io/nixvim/colorschemes/tokyonight/index.html
                    style = "night";
                };
            };
        };
    };
}
