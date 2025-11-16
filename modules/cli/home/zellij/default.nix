{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.cli.zellij.enable {
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
        keybinds = {
          normal = {
            "bind \"Enter\"" = {
              # Passthru `Enter`.
              WriteChars = "\\u{000D}";
              # Invoke autolock to immediately assess proper lock state.
              # (This provides a snappier experience compared to
              # solely relying on `reaction_seconds` to elapse.)
              "MessagePlugin \"autolock\"" = { };
            };
          };
          locked = {
            "bind \"Ctrl g\"" = {
              "MessagePlugin \"autolock\"" = {
                payload = "toggle";
              }; # toggle the autolock plugin.
              SwitchToMode = "Normal";
            };
          };
          "shared_except \"locked\"" = {
            "bind \"Ctrl g\"" = {
              "MessagePlugin \"autolock\"" = {
                payload = "toggle";
              }; # Disable the autolock plugin.
              SwitchToMode = "Locked";
            };
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
            triggers = "nvim|vim";
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
