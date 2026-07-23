{
  lib,
  config,
  ...
}:
let
  palette = import ./theme/palette.nix;
in
{
  config = lib.mkIf config.home.desktop.hyprland.dunst.enable {
    services.dunst = {
      enable = true;
      settings = {
        global = {
          # Position
          origin = "top-right";
          offset = "(16, 56)"; # 56px down to clear the waybar
          notification_limit = 5;
          gap_size = 8;

          # Dimensions
          width = "(200, 380)";
          height = "(0, 300)";
          padding = 14;
          horizontal_padding = 16;
          text_icon_padding = 12;

          # Appearance
          corner_radius = 12;
          frame_width = 1;
          separator_height = 1;
          separator_color = "frame";

          # Font
          font = "JetBrainsMono Nerd Font Propo 11";

          # Text
          format = "<b>%s</b>\\n%b";
          alignment = "left";
          vertical_alignment = "center";
          word_wrap = true;
          markup = "full";
          ellipsize = "end";
          show_age_threshold = 60;
          ignore_newline = false;
          stack_duplicates = true;
          hide_duplicate_count = false;

          # Behaviour
          timeout = 6;
          idle_threshold = 120;
          sticky_history = true;
          history_length = 50;
          always_run_script = true;

          # Icons
          icon_theme = "rose-pine";
          enable_recursive_icon_lookup = true;
          icon_position = "left";
          min_icon_size = 24;
          max_icon_size = 48;

          # Progress bar
          progress_bar = true;
          progress_bar_height = 6;
          progress_bar_frame_width = 0;
          progress_bar_min_width = 150;
          progress_bar_max_width = 300;
          progress_bar_corner_radius = 3;
          highlight = palette.mauve;
        };

        urgency_low = {
          background = "${palette.crust}99";
          foreground = palette.subtext0;
          frame_color = "${palette.overlay0}40";
          timeout = 4;
        };

        urgency_normal = {
          background = "${palette.crust}99";
          foreground = palette.text;
          frame_color = "${palette.mauve}73";
          timeout = 6;
        };

        urgency_critical = {
          background = "${palette.crust}b3";
          foreground = palette.red;
          frame_color = "${palette.red}b3";
          timeout = 0; # never auto-dismiss
        };
      };
    };
  };
}
