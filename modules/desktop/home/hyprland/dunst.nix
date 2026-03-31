{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    services.dunst = {
      enable = true;
      settings = {
        global = {
          # Position
          origin = "top-right";
          offset = "16x56"; # 56px down to clear the waybar
          notification_limit = 5;
          gap_size = 8;

          # Dimensions
          width = "(200, 380)";
          height = 120;
          padding = 14;
          horizontal_padding = 16;
          text_icon_padding = 12;
          icon_size = 32;

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
          history_length = 20;
          always_run_script = true;
          close = "ctrl+space";
          close_all = "ctrl+shift+space";

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
        };

        urgency_low = {
          background = "#1e1e2e"; # base
          foreground = "#a6adc8"; # subtext0
          frame_color = "#313244"; # surface0
          timeout = 4;
        };

        urgency_normal = {
          background = "#1e1e2e"; # base
          foreground = "#cdd6f4"; # text
          frame_color = "#cba6f7"; # mauve
          timeout = 6;
        };

        urgency_critical = {
          background = "#1e1e2e"; # base
          foreground = "#f38ba8"; # red
          frame_color = "#f38ba8"; # red
          timeout = 0; # never auto-dismiss
        };
      };
    };
  };
}
