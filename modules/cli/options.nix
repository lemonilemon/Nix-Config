{
  lib,
  config,
  ...
}:
{
  options = {
    home.cli = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.enable && config.cli.enable;
        description = "Enable my CLI settings, including configurations for git, nvim, various programs, shells, and zellij";
      };

      git = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli.enable;
          description = "Enable git settings";
        };
      };

      nvim = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli.enable;
          description = "Enable nvim settings";
        };
      };

      programs = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli.enable;
          description = "Enable my CLI programs settings";
        };
      };

      shells = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli.enable;
          description = "Enable shell configuration";
        };

        zoxide = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.cli.shells.enable;
            description = "Enable zoxide";
          };
        };

        starship = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.cli.shells.enable;
            description = "Enable starship prompt";
          };
        };

        zsh = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.cli.shells.enable;
            description = "Enable zsh shell settings";
          };
        };
      };

      multiplexer = {
        program = lib.mkOption {
          type = lib.types.enum [
            "tmux"
            "zellij"
          ];
          default = "tmux";
          description = "Choose your terminal multiplexer";
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

    nixos.cli = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.enable && config.cli.enable;
        description = "Enable my CLI settings for NixOS modules";
      };
    };
  };

}
