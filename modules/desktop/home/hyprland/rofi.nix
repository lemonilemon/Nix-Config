{
  lib,
  config,
  pkgs,
  ...
}:
let
  palette = import ./theme/palette.nix;

  inherit (config.lib.formats.rasi) mkLiteral;

  # rasi takes #RRGGBBAA. The two-digit alphas reuse the family's translucency
  # levels: cc = 0.80 (hyprlock inner_color), b3 = 0.70 (panel glass),
  # 9e = 0.62 (wlogout rest border), 33 = 0.20 (wlogout hover wash).
  withAlpha = hex: a: mkLiteral "${hex}${a}";
  solid = mkLiteral;
in
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    # Same opt-out as hyprlock.nix and wlogout/: the catppuccin module points
    # programs.rofi.theme at its stock rasi, which both collides with the theme
    # below (two definitions of the same option) and reads as foreign next to
    # the session-overlay family.
    catppuccin.rofi.enable = false;

    # Windows switcher / App launcher.
    #
    # Third member of the session-overlay family. hyprlock's input field and
    # wlogout's tiles set the vocabulary -- surface0 glass controls, 2px
    # borders, blue as the interactive accent -- and selection reuses wlogout's
    # hover formula: a 20% accent wash plus accent-coloured text. The blurred
    # backdrop that completes the glass comes from the compositor; see the
    # `rofi` layerrule in ./default.nix.
    programs.rofi = {
      enable = true;
      package = pkgs.rofi;
      font = "JetBrains Mono Nerd Font 14";
      extraConfig = {
        # rofi 2.0 renamed modi -> modes and combi-modi -> combi-modes; the old
        # spellings are dropped silently. No ssh mode: remote work goes through
        # sesh-managed tmux sessions (cli/home/tmux/sesh.nix), not bare ssh.
        modes = "window,drun,run";
        combi-modes = "drun,run";
        show-icons = true;
        matching = "fuzzy";
        terminal = "kitty";
        drun-match-fields = "name";
        drun-display-format = "{name}";
        kb-cancel = "Escape";
        sidebar-mode = true;
        # Mode names shown in the switcher. Material Design Icons from the same
        # Nerd Font as wlogout's tiles, so the stroke weight matches across the
        # family.
        display-drun = "󰀻 Apps"; # U+F003B md-apps
        display-run = "󰆍 Run"; # U+F018D md-console
        display-window = "󰖯 Windows"; # U+F05AF md-window-maximize
      };
      theme = {
        "*" = {
          background-color = mkLiteral "transparent";
          text-color = solid palette.text;
        };

        # The panel is the tier's container: wlogout's scrim thickened to
        # carry a list, wearing a tile's border. Blue, like hyprlock's input
        # outline -- the family's one interactive accent.
        window = {
          width = mkLiteral "640px";
          background-color = withAlpha palette.base "b3";
          border = mkLiteral "2px solid";
          border-color = withAlpha palette.blue "9e";
          border-radius = mkLiteral "16px"; # = decoration.rounding
          padding = mkLiteral "20px";
        };

        mainbox = {
          spacing = mkLiteral "12px";
          children = map mkLiteral [
            "inputbar"
            "listview"
            "mode-switcher"
          ];
        };

        # hyprlock's input field, minus its border: inside an already-bordered
        # panel a second outline would double the line.
        inputbar = {
          background-color = withAlpha palette.surface0 "cc";
          border-radius = mkLiteral "12px"; # (hyprlock) rounding
          padding = mkLiteral "12px 16px";
          spacing = mkLiteral "10px";
          children = map mkLiteral [
            "textbox-search"
            "entry"
          ];
        };

        # In-field glyph, as hyprlock's 󰌾 lives inside its field.
        textbox-search = {
          expand = false;
          content = "󰍉"; # U+F0349 md-magnify
          text-color = solid palette.blue;
        };

        entry = {
          placeholder = "Search";
          placeholder-color = solid palette.subtext0;
        };

        listview = {
          lines = 8;
          fixed-height = false;
          scrollbar = false;
          spacing = mkLiteral "4px";
        };

        element = {
          padding = mkLiteral "8px 12px";
          spacing = mkLiteral "10px";
          border-radius = mkLiteral "12px";
        };

        element-icon = {
          size = mkLiteral "1.5em";
        };

        element-text = {
          text-color = mkLiteral "inherit";
          vertical-align = mkLiteral "0.5";
          highlight = mkLiteral "bold";
        };

        # Urgent/active only appear in the window mode; red/green follow
        # hyprlock's fail_color/check_color so state colours mean the same
        # thing everywhere.
        "element urgent" = {
          text-color = solid palette.red;
        };
        "element active" = {
          text-color = solid palette.green;
        };

        "element selected.normal" = {
          background-color = withAlpha palette.blue "33";
          text-color = solid palette.blue;
        };
        "element selected.urgent" = {
          background-color = withAlpha palette.red "33";
          text-color = solid palette.red;
        };
        "element selected.active" = {
          background-color = withAlpha palette.green "33";
          text-color = solid palette.green;
        };

        mode-switcher = {
          spacing = mkLiteral "8px";
        };

        button = {
          padding = mkLiteral "8px";
          border-radius = mkLiteral "12px";
          text-color = solid palette.subtext0;
        };

        "button selected" = {
          background-color = withAlpha palette.blue "33";
          text-color = solid palette.blue;
        };

        # Error/status box (e.g. "no matches"), dressed as the input field.
        message = {
          background-color = withAlpha palette.surface0 "cc";
          border-radius = mkLiteral "12px";
          padding = mkLiteral "10px 12px";
        };
      };
    };
  };
}
