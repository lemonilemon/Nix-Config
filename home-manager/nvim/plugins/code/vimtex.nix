{ pkgs
, ...
}: {
  programs.zathura = {
    enable = true;
  };
  programs.nixvim = {
    globals = {
      # tex flavor
      tex_flavor = "latex";
    };
    # https://nix-community.github.io/nixvim/plugins/vimtex/index.html
    plugins = {
      vimtex = {
        enable = true;
        texlivePackage = pkgs.texlive.combined.scheme-full;
        settings = {
          view_method = "zathura";
          compiler_latexmk_engines = {
            _ = "-pdflatex";
          };
          tex_comment_nospell = 1;
          compiler_progname = "nvr";
          view_general_options = "--unique file:@pdf\#src:@line@tex";
        };
      };
    };
  };
}
