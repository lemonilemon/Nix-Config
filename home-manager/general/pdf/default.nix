{
  pkgs,
  lib,
  ...
}:
{
  home.packages = with pkgs; [
    # tex & markdown
    texlive.combined.scheme-full # tex
    multimarkdown # markdown

    # file conversion
    pandoc

  ];
  # pdf viewer
  programs.zathura = {
    enable = lib.mkDefault true;
    extraConfig = ''
      # Zathura configuration file
      # See man `man zathurarc'
      map r reload
      map p print
      map i recolor
      map J zoom out
      map K zoom in
      map R rotate
      unmap f
      map f toggle_fullscreen
      map [fullscreen] f toggle_fullscreen

      # Open document in fit-width mode by default
      set adjust-open "best-fit"

      # One page per row by default
      set pages-per-row 1

      #stop at page boundries
      set scroll-page-aware "true"
      set scroll-full-overlap 0.01
      set scroll-step 100

      #zoom settings
      set zoom-min 10
      set guioptions ""

      set font "FiraCode 15"

      # zathurarc-dark
      set notification-error-bg       "#586e75" # base01  # seem not work
      set notification-error-fg       "#dc322f" # red
      set notification-warning-bg     "#586e75" # base01
      set notification-warning-fg     "#dc322f" # red
      set notification-bg             "#586e75" # base01
      set notification-fg             "#b58900" # yellow

      set completion-group-bg         "#002b36" # base03
      set completion-group-fg         "#839496" # base0
      set completion-bg               "#073642" # base02
      set completion-fg               "#93a1a1" # base1
      set completion-highlight-bg     "#586e75" # base01
      set completion-highlight-fg     "#eee8d5" # base2

      # Define the color in index mode
      set index-bg                   "#073642" # base02
      set index-fg                   "#93a1a1" # base1
      set index-active-bg             "#586e75" # base01
      set index-active-fg             "#eee8d5" # base2

      set inputbar-bg                 "#586e75" # base01
      set inputbar-fg                 "#eee8d5" # base2

      set statusbar-bg                "#073642" # base02
      set statusbar-fg                "#93a1a1" # base1

      set highlight-color             "#657b83" # base00  # hightlight match when search keyword(vim's /)
      set highlight-active-color      "#268bd2" # blue

      set default-bg                  "#073642" # base02
      set default-fg                  "#93a1a1" # base1
      set render-loading              true
      set render-loading-fg           "#073642" # base02
      set render-loading-bg           "#073642" # base02

      # Recolor book content's color
      set recolor                     true
      set recolor-lightcolor          "#073642" # base02
      set recolor-darkcolor           "#93a1a1" # base1
      set recolor-keephue             true      # keep original color
    '';

  };

  programs.sioyek = {
    enable = lib.mkDefault true;

    # === CATPPUCCIN MOCHA === #
    config = {
      "background_color" = "#1e1e2e";

      "text_highlight_color" = "#f9e2af";
      "visual_mark_color" = "#7f849c";

      "search_highlight_color" = "#f9e2af";
      "link_highlight_color" = "#89b4fa";
      "synctex_highlight_color" = "#a6e3a1";

      "highlight_color_a" = "#f9e2af";
      "highlight_color_b" = "#a6e3a1";
      "highlight_color_c" = "#89dceb";
      "highlight_color_d" = "#eba0ac";
      "highlight_color_e" = "#cba6f7";
      "highlight_color_f" = "#f38ba8";
      "highlight_color_g" = "#f9e2af";

      "custom_background_color" = "#1e1e2e";
      "custom_text_color" = "#cdd6f4";

      "ui_text_color" = "#cdd6f4";
      "ui_background_color" = "#313244";
      "ui_selected_text_color" = "#cdd6f4";
      "ui_selected_background_color" = "#585b70";
    };
  };
}
