{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.cli.programs.enable {
    home.packages = with pkgs; [
      # archives
      zip
      unzip
      _7zz # 7zip
      # utils
      netcat
      direnv
      nix-direnv
      gnumake
      curl
      wget
      rqbit # a lightweight bittorrent client
      ffmpeg

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
