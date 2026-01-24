{
  lib,
  config,
  osConfig ? null,
  helpers,
  ...
}:
{
  options = {
    home.cli = {
      enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.cli.enable";
        default = config.home.enable && config.cli.enable;
        description = "Enable my CLI settings, including configurations for git, nvim, various programs, shells, and multiplexers";
      };

      git = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.cli.git.enable";
          default = config.home.cli.enable;
          description = "Enable git settings";
        };
      };

      nvim = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.cli.nvim.enable";
          default = config.home.cli.enable;
          description = "Enable nvim settings";
        };
      };

      programs = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.cli.programs.enable";
          default = config.home.cli.enable;
          description = "Enable my CLI programs settings";
        };
      };

      shells = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.cli.shells.enable";
          default = config.home.cli.enable;
          description = "Enable shell configuration";
        };

        zoxide = {
          enable = helpers.mkHomeOpt {
            inherit osConfig;
            path = "home.cli.shells.zoxide.enable";
            default = config.home.cli.shells.enable;
            description = "Enable zoxide";
          };
        };

        starship = {
          enable = helpers.mkHomeOpt {
            inherit osConfig;
            path = "home.cli.shells.starship.enable";
            default = config.home.cli.shells.enable;
            description = "Enable starship prompt";
          };
        };

        zsh = {
          enable = helpers.mkHomeOpt {
            inherit osConfig;
            path = "home.cli.shells.zsh.enable";
            default = config.home.cli.shells.enable;
            description = "Enable zsh shell settings";
          };
        };
      };

      multiplexer = {
        program = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.cli.multiplexer.program";
          default = "tmux";
          type = lib.types.enum [
            "tmux"
            "zellij"
          ];
          description = "Which terminal multiplexer to use";
        };
        zellij.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli.multiplexer.program == "zellij";
          description = "Enable zellij multiplexer";
        };
        tmux.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli.multiplexer.program == "tmux";
          description = "Enable tmux multiplexer";
        };
      };
    };
  };

}
