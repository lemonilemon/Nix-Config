{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.cli.multiplexer.tmux.enable {
    programs.tmux = {
      enable = true;
      clock24 = true; # 24-hour clock in status bar
      baseIndex = 1; # Start windows at 1 (matches keyboard order)
      escapeTime = 0; # Fixes delay issues in Vim/Neovim
      keyMode = "vi"; # Vi-style navigation in copy mode
      mouse = true; # Enable mouse support (clickable windows/panes)
      sensibleOnTop = true; # Sensible defaults

      extraConfig = ''
        set -g allow-passthrough on
        set -g set-clipboard on
        set -g detach-on-destroy off
        set -g renumber-windows on

        # Status bar customization
        set -g status-position top
        set -g status-right-length 100
        set -g status-left-length 100
        set -g status-left ""

        set -ga update-environment TERM
        set -ga update-environment TERM_PROGRAM

        # Split panes using | and - (more visual than % and ")
        unbind %
        unbind '"'
        bind | split-window -h -c "#{pane_current_path}"
        bind - split-window -v -c "#{pane_current_path}"

        # Keep new windows in the current path
        bind c new-window -c "#{pane_current_path}"

        # Vim like pane navigation with Alt + h/j/k/l
        # bind -n M-h select-pane -L
        # bind -n M-j select-pane -D
        # bind -n M-k select-pane -U
        # bind -n M-l select-pane -R

        # Reload config file
        bind r source-file ~/.config/tmux/tmux.conf \; display-message "Config reloaded!"
      '';

      plugins = with pkgs; [
        {
          # Theme: Catppuccin (Modern, clean, similar vibes to Zellij)
          plugin = tmuxPlugins.catppuccin;
          extraConfig = ''
            set -g @catppuccin_flavour 'mocha' 
            set -g @catppuccin_window_tabs_enabled on
            set -g @catppuccin_window_status_style "rounded"
            set -g @catppuccin_date_time "%H:%M"
            set -g status-right "#{E:@catppuccin_status_application}"
            set -ag status-right "#{E:@catppuccin_status_session}"
            set -ag status-right "#{E:@catppuccin_status_uptime}"
          '';
        }
        {
          # Navigation: Navigate between Vim and Tmux panes seamlessly
          plugin = tmuxPlugins.vim-tmux-navigator;
          extraConfig = ''
            set -g @vim_navigator_mapping_left "M-h"
            set -g @vim_navigator_mapping_right "M-l"
            set -g @vim_navigator_mapping_up "M-k"
            set -g @vim_navigator_mapping_down "M-j"
            set -g @vim_navigator_mapping_prev ""  # removes the C-\ binding
            set -g @vim_navigator_prefix_mapping_clear_screen ""
          '';
        }
        {
          # Persistence: Auto-save sessions (Resurrect + Continuum)
          plugin = tmuxPlugins.resurrect;
          extraConfig = "set -g @resurrect-strategy-nvim 'session'";
        }
        {
          plugin = tmuxPlugins.continuum;
          extraConfig = ''
            set -g @continuum-restore 'on'
            set -g @continuum-save-interval '60' # Save every 60 minutes
          '';
        }
        # Better copy/paste behavior
        tmuxPlugins.yank
        # which-key style popup for tmux commands
        {
          plugin = tmuxPlugins.tmux-which-key;
          extraConfig = ''
            set -g @tmux-which-key-xdg-enable 1
            set -g @tmux-which-key-disable-autobuild 1
          '';
        }
      ];

    };
    programs.zsh = {
      enable = true;
      localVariables = {
        ZSH_TMUX_AUTOSTART = true;
        ZSH_TMUX_AUTOCONNECT = true; # Connects to an existing session if one exists
        ZSH_TMUX_FIXTERM = true; # Fixes TERM variable issues when launching tmux from zsh
        ZSH_TMUX_UNICODE = true; # Enables better Unicode support in tmux
      };
      zplug.plugins = [
        {
          name = "plugins/tmux";
          tags = [ "from:oh-my-zsh" ];
        }
      ];
    };
  };
}
