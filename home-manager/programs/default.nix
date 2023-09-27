{ pkgs, ... }:
{
    home.packages = with pkgs; [
        # archives
        zip
        unzip

        # utils
        bat
        netcat
        neofetch
        ripgrep
        direnv

        # package manager
        yarn

        # finder / viewer
        fd
        fzf
        tree
    ];
    programs = {
        bat = {
            enable = true;
            config = {
                pager = "less -FR";
                theme = "Catppuccin-mocha";
            };
            themes = {
                Catppuccin-mocha = builtins.readFile (pkgs.fetchurl {
                    url = "https://raw.githubusercontent.com/catppuccin/bat/main/Catppuccin-mocha.tmTheme";
                    hash = "sha256-qMQNJGZImmjrqzy7IiEkY5IhvPAMZpq0W6skLLsng/w=";
                });
            };
        };
    };
}
