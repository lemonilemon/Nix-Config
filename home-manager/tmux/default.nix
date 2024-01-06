{
    pkgs,
    ...
}: {
    programs.tmux = {
        enable = true;
        clock24 = true;
        keyMode = "vi";
        mouse = true;
        baseIndex = 1;
        terminal = "screen-256color";
        prefix = "C-Space";
        plugins = with pkgs; [
            tmuxPlugins.sensible
            tmuxPlugins.yank 
            tmuxPlugins.battery
            {
                plugin = tmuxPlugins.catppuccin;
                extraConfig = ''
                    set -g @catppuccin_flavour "mocha"
                    set -g @catppuccin_window_left_separator "█"
                    set -g @catppuccin_window_right_separator "█ "
                    set -g @catppuccin_window_number_position "right"
                    set -g @catppuccin_window_middle_separator "  █"

                    set -g @catppuccin_window_default_fill "number"

                    set -g @catppuccin_window_current_fill "number"

                    set -g @catppuccin_status_modules_left "application session date_time"
                    set -g @catppuccin_status_modules_right "application session date_time"
                    set -g @catppuccin_status_left_separator  ""
                    set -g @catppuccin_status_right_separator " "
                    set -g @catppuccin_status_right_separator_inverse "yes"
                    set -g @catppuccin_status_fill "all"
                    set -g @catppuccin_status_connect_separator "no"
                '';
            }
        ];
        extraConfig = ''            
            
            # Vim style pane selection
            bind h select-pane -L
            bind j select-pane -D 
            bind k select-pane -U
            bind l select-pane -R
             
            # Use Alt-arrow keys without prefix key to switch panes
            bind -n M-Left select-pane -L
            bind -n M-Right select-pane -R
            bind -n M-Up select-pane -U
            bind -n M-Down select-pane -D

            # Shift arrow to switch windows
            bind -n S-Left  previous-window
            bind -n S-Right next-window

            # Shift Alt vim keys to switch windows
            bind -n M-H previous-window
            bind -n M-L next-window
        '';
    };
}
