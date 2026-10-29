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

        atuin.threads.enable = lib.mkOption {
          type = lib.types.bool;
          default = false;
          description = "Capture AI coding agents' sessions as Atuin threads, synced like shell history";
        };

        atuin.mcp.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli.programs.atuin.threads.enable;
          description = "Give Claude Code, Codex and pi Atuin's MCP server, to search shell history and past threads";
        };

        opencode.enable = lib.mkOption {
          type = lib.types.bool;
          default = false;
          description = "Enable opencode and its multi-agent setup";
        };

        aerc.enable = lib.mkOption {
          type = lib.types.bool;
          default = false;
          description = "Enable the aerc mail client";
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
