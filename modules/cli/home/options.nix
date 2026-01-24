{
  lib,
  config,
  osConfig,
  ...
}:
{
  options = {
    home.cli = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default =
          if osConfig == null then config.home.enable && config.cli.enable else osConfig.home.cli.enable;
        description = "Enable my CLI settings, including configurations for git, nvim, various programs, shells, and multiplexers";
      };

      git = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = if osConfig == null then config.home.cli.enable else osConfig.home.cli.git.enable;
          description = "Enable git settings";
        };
      };

      nvim = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = if osConfig == null then config.home.cli.enable else osConfig.home.cli.nvim.enable;
          description = "Enable nvim settings";
        };
      };

      programs = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = if osConfig == null then config.home.cli.enable else osConfig.home.cli.programs.enable;
          description = "Enable my CLI programs settings";
        };
      };

      shells = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = if osConfig == null then config.home.cli.enable else osConfig.home.cli.shells.enable;
          description = "Enable shell configuration";
        };

        zoxide = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default =
              if osConfig == null then config.home.cli.shells.enable else osConfig.home.cli.shells.zoxide.enable;
            description = "Enable zoxide";
          };
        };

        starship = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default =
              if osConfig == null then
                config.home.cli.shells.enable
              else
                osConfig.home.cli.shells.starship.enable;
            description = "Enable starship prompt";
          };
        };

        zsh = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default =
              if osConfig == null then config.home.cli.shells.enable else osConfig.home.cli.shells.zsh.enable;
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
          default = if osConfig == null then "tmux" else osConfig.home.cli.multiplexer.program;
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
