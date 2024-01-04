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
        plugins = with pkgs; [
            tmuxPlugins.yank 
            {
                plugin = tmuxPlugins.catppuccin;
                extraConfig = "set -g @catppuccin_flavour \'mocha\'";
            }
            tmuxPlugins.sensible
        ];
        extraConfig = ''
            set -g default-terminal screen-256color

            # use C-Space
            unbind C-b
            set -g prefix C-Space
            bind C-Space send-prefix

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
