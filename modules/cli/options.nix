{
  lib,
  config,
  ...
}:
{
  options = {
    home.cli-settings = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.enable && config.cli-settings.enable;
        description = "Enable my CLI settings, including configurations for git, nvim, various programs, shells, and zellij";
      };

      git = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli-settings.enable;
          description = "Enable git settings";
        };
      };

      nvim = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli-settings.enable;
          description = "Enable nvim settings";
        };
      };

      programs = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli-settings.enable;
          description = "Enable my CLI programs settings";
        };
      };

      shells = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.cli-settings.enable;
          description = "Enable shell configuration";
        };

        zoxide = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.cli-settings.shells.enable;
            description = "Enable zoxide";
          };
        };

        starship = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.cli-settings.shells.enable;
            description = "Enable starship prompt";
          };
        };

        zsh = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.cli-settings.shells.enable;
            description = "Enable zsh shell settings";
          };
        };
      };

      zellij.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.cli-settings.enable;
        description = "Enable zellij multiplexer";
      };
    };

    nixos.cli-settings = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.enable && config.cli-settings.enable;
        description = "Enable my CLI settings for NixOS modules";
      };
    };
  };

}
