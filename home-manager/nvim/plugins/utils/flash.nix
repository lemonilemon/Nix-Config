{
    programs.nixvim = {
        # https://nix-community.github.io/nixvim/plugins/flash/index.html
        plugins.flash = {
            enable = true;
            settings = {
                label = {
                    uppercase = false;
                };
                labels = "asdfghjklqwertyuiopzxcvbnm";
                search = {
                    mode = "fuzzy";
                };
                jump = {
                    autojump = true;
                };
            };
        };
        keymaps = [
            {
                mode = [
                    "n"
                    "x"
                    "o"
                ];
                key = "s";
                action = "<cmd>lua require('flash').jump()<cr>";
                options = {
                    desc = "Flash";
                };
            }

            {
                mode = [
                    "n"
                    "x"
                    "o"
                ];
                key = "S";
                action = "<cmd>lua require('flash').treesitter()<cr>";
                options = {
                    desc = "Flash Treesitter";
                };
            }

            {
                mode = "o";
                key = "r";
                action = "<cmd>lua require('flash').remote()<cr>";
                options = {
                    desc = "Remote Flash";
                };
            }

            {
                mode = [
                    "x"
                    "o"
                ];
                key = "R";
                action = "<cmd>lua require('flash').treesitter_search()<cr>";
                options = {
                    desc = "Treesitter Search";
                };
            }
        ];
    };
}
