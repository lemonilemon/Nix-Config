{ pkgs, ... }:
{
    imports = [
    ];

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
        nix-direnv
        gnumake
        curl
        wget
        jq
        psmisc

    # package manager
    yarn

    # finder / viewer
    fd
    fzf
    tree

    # pdf reader
    zathura
    # note taking
    obsidian
  ];
  programs = {
    bat = {
      enable = true;
      themes = {
        Catppuccin-mocha = {
          src = pkgs.fetchFromGitHub {
            owner = "catppuccin";
            repo = "bat";
            rev = "ba4d16880d63e656acced2b7d4e034e4a93f74b1";
            sha256 = "6WVKQErGdaqb++oaXnY3i6/GuH2FhTgK0v4TN4Y0Wbw=";
          };
          file = "Catppuccin-mocha.tmTheme";
        };
      };
      config = {
        pager = "less -FR";
        theme = "Catppuccin-mocha";
      };
    };

    direnv = {
      enable = true;
      enableZshIntegration = true;
      nix-direnv.enable = true;
    };

    fzf = {
      enable = true;
      enableZshIntegration = true;
      tmux.enableShellIntegration = true;
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
        completion-highlight-fg = "#4C4F69";
        completion-group-bg = "#CCD0DA";
        completion-group-fg = "#1E66F5";

        statusbar-fg = "#4C4F69";
        statusbar-bg = "#CCD0DA";

        notification-bg = "#CCD0DA";
        notification-fg = "#4C4F69";
        notification-error-bg = "#CCD0DA";
        notification-error-fg = "#D20F39";
        notification-warning-bg = "#CCD0DA";
        notification-warning-fg = "#FAE3B0";

        inputbar-fg = "#4C4F69";
        inputbar-bg = "#CCD0DA";

        recolor-lightcolor = "#EFF1F5";
        recolor-darkcolor = "#4C4F69";

        index-fg = "#4C4F69";
        index-bg = "#EFF1F5";
        index-active-fg = "#4C4F69";
        index-active-bg = "#CCD0DA";

        render-loading-bg = "#EFF1F5";
        render-loading-fg = "#4C4F69";

        highlight-color = "#575268";
        highlight-fg = "#EA76CB";
        highlight-active-color = "#EA76CB";
        recolor-reverse-video = "true";
        recolor-keephue = "true";
      };
    };
  };
}
