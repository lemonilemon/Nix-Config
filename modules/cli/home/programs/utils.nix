{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.cli-settings.programs.enable {
    home.packages = with pkgs; [
      # archives
      zip
      unzip
      # utils
      netcat
      direnv
      nix-direnv
      gnumake
      curl
      wget

      # command alternatives
      lsd # ls
      bat # cat
      fd # find
      jq # json parser
      ripgrep # grep
    ];
    programs = {
      # command alternatives
      lsd = {
        enable = true;
        enableZshIntegration = true;
      };
      bat = {
        enable = true;
        config = {
          pager = "less -FR";
          # Theme controlled by catppuccin
          # theme = "Catppuccin-mocha";
        };
      };
      ripgrep.enable = true;
      fd.enable = true;
    };

  };
}
