{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.cli.programs.enable {
    programs.yazi = {
      enable = true;
      enableZshIntegration = true;
      extraPackages = with pkgs; [
        glow
        ouch
      ];
      plugins = {
        piper = pkgs.yaziPlugins.piper;
        ouch = pkgs.yaziPlugins.ouch;
        git = pkgs.yaziPlugins.git;
      };

      settings = {
        mgr = {
          show_hidden = true;
          sort_dir_first = true;
        };
        plugin = {
          prepend_previewers = [
            {
              url = "*.md";
              run = "piper -- CLICOLOR_FORCE=1 glow -w=$w -s=dark \"$1\"";
            }
            # Use default previewer
            # {
            #   mime = "application/{*zip,tar,bzip2,7z*,rar,xz,zstd,java-archive}";
            #   run = "ouch";
            # }
          ];
        };
        # opener = {
        #   glow = [
        #     {
        #       run = "glow -p \"$@\"";
        #       block = true;
        #       desc = "View with Glow";
        #     }
        #   ];
        # };
        #
        # open = {
        #   prepend_rules = [
        #     {
        #       name = "*.md";
        #       use = "glow";
        #     }
        #   ];
        # };
      };
    };
  };
}
