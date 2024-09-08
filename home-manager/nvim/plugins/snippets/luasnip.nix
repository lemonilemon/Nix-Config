{ pkgs, ... }:
{
    programs.nixvim.plugins = {
        # https://nix-community.github.io/nixvim/plugins/luasnip/index.html
        luasnip = {
            enable = true;
            settings = {
                enable_autosnippets = true;
                store_selection_keys = "<Tab>";
            };
            fromVscode = [
                {
                    lazyLoad = true;
                    paths = "${pkgs.vimPlugins.friendly-snippets}";
                }
            ];
        };
    };
}
