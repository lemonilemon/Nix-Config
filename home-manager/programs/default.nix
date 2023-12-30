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

        # pdf reader
        zathura

        # git
        lazygit

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
        zathura = {
            enable = true;
            options = {
                # Open document in fit-width mode by default
                adjust-open = "best-fit";
                window-height = 1000;
                window-width = 1600;

                # One page per row by default
                pages-per-row = 1;

                #stop at page boundries
                scroll-page-aware = "true";
                scroll-full-overlap = 1;
                scroll-step = 100;

                #zoom settings
                zoom-min = 10;
                guioptions = "";

                # cattppuccin-latte 
                default-bg = "#EFF1F5";
                completion-bg = "#CCD0DA";
                completion-fg = "#4C4F69";
                completion-highlight-bg = "#575268";
                completion-highlight-fg	= "#4C4F69";
                completion-group-bg	= "#CCD0DA";
                completion-group-fg = "#1E66F5";

                statusbar-fg = "#4C4F69";
                statusbar-bg = "#CCD0DA";

                notification-bg	= "#CCD0DA";
                notification-fg = "#4C4F69";
                notification-error-bg = "#CCD0DA";
                notification-error-fg = "#D20F39";
                notification-warning-bg	= "#CCD0DA";
                notification-warning-fg	= "#FAE3B0";

                inputbar-fg = "#4C4F69";
                inputbar-bg = "#CCD0DA";

                recolor-lightcolor = "#EFF1F5";
                recolor-darkcolor = "#4C4F69";

                index-fg = "#4C4F69";
                index-bg = "#EFF1F5";
                index-active-fg = "#4C4F69";
                index-active-bg	= "#CCD0DA";

                render-loading-bg = "#EFF1F5";
                render-loading-fg = "#4C4F69";

                highlight-color = "#575268";
                highlight-fg = "#EA76CB";
                highlight-active-color = "#EA76CB";
                recolor-reverse-video = "true";
                recolor-keephue = "true";
            };
            #mappings = {
                #f  = "toggle_fullscreen";
                #"[fullscreen]f"  = "toggle_fullscreen";
            #};
        };
    };
}
