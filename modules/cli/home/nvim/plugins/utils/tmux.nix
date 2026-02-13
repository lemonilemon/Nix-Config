{
  config,
  lib,
  ...
}:
{
  config = lib.mkIf (config.home.cli.multiplexer.program == "tmux") {
    programs.nixvim = {
      plugins = {
        tmux-navigator = {
          enable = true;
          settings = {
            preserve_zoom = 1;
            save_on_switch = 1;
          };
          keymaps = [
            # Move to different zellij pane using Alt + hjkl
            {
              action = "left";
              key = "<A-h>";
            }
            {
              action = "down";
              key = "<A-j>";
            }
            {
              action = "up";
              key = "<A-k>";
            }
            {
              action = "right";
              key = "<A-l>";
            }
          ];
        };
      };
    };
  };
}
