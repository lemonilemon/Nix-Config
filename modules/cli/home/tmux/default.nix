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

      # 2. The Solution to your "Graphics Protocol" issue
      # "allow-passthrough" enables Yazi/Neovim images to render directly
      extraConfig = ''
        set -g allow-passthrough on
        set -ga update-environment TERM
        set -ga update-environment TERM_PROGRAM

        # Split panes using | and - (more visual than % and ")
        bind | split-window -h -c "#{pane_current_path}"
        bind - split-window -v -c "#{pane_current_path}"

        # Keep new windows in the current path
        bind c new-window -c "#{pane_current_path}"
      '';

      # 3. Recommended Plugins (Installed via Nix, no TPM needed)
      plugins = with pkgs; [
        {
          # Theme: Catppuccin (Modern, clean, similar vibes to Zellij)
          plugin = tmuxPlugins.catppuccin;
          extraConfig = ''
            set -g @catppuccin_flavour 'mocha' 
            set -g @catppuccin_window_tabs_enabled on
            set -g @catppuccin_date_time "%H:%M"
          '';
        }
        {
          # Navigation: Navigate between Vim and Tmux panes seamlessly
          # (Matches Zellij's directional navigation)
          plugin = tmuxPlugins.vim-tmux-navigator;
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
