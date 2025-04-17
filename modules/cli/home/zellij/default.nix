{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.cli-settings.zellij.enable {
    programs.zellij = {
      enable = true;
      enableZshIntegration = true;
      # attachExistingSession = true;
      exitShellOnExit = true;
      # For KDL settings in Nix, see: https://github.com/nix-community/home-manager/blob/master/modules/lib/generators.nix
      settings = {
        show_startup_tips = false;
        theme = "catppuccin-mocha";
        default_shell = "zsh";
        ui = {
          pane_frames = {
            rounded_corners = true;
          };
        };
        plugins = {
          autolock = {
            # Props for KDL
            _props = {
              location = "file:~/.config/zellij/plugins/zellij-autolock.wasm";
            };
            # Enabled at start?
            is_enabled = true;
            # Lock when any open these programs open.
            triggers = "nvim|vim|git|fzf|zoxide|atuin";
            # Reaction to input occurs after this many seconds. (default=0.3)
            #(An existing scheduled reaction prevents additional reactions.)
            reaction_seconds = "0.3";
            # Print to Zellij log? (default=false)
            print_to_log = true;
          };
        };
        load_plugins = {
          autolock = [ ];
        };
      };
    };
    home.file.".config/zellij/plugins" = {
      enable = true;
      recursive = true;
      source = ./plugins;
    };
  };
}
