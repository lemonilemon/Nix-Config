{
  pkgs,
  ...
}:
{
  programs.nixvim = {
    extraPlugins = with pkgs.vimUtils; [
      (buildVimPlugin {
        pname = "template.nvim";
        version = "2024-09-08";
        src = pkgs.fetchFromGitHub {
          owner = "nvimdev";
          repo = "template.nvim";
          rev = "main";
          sha256 = "sha256-SsTqdOve0uAP9fApBSVIUj0JIOjneQD02CXbA0dRCWo=";
        };
      })
    ];
    extraConfigLua = ''
      require("telescope").load_extension('find_template')

      require("template").setup({
        temp_dir = "~/.config/nvim/template",
        author = "lemonilemon",
      })
    '';
    # Template files
    # The template of C
    extraFiles."template/c/Basic.tpl".source = ./template/c/Basic.tpl;
    # The template of C++
    extraFiles."template/cpp/Basic.tpl".source = ./template/cpp/Basic.tpl;
    extraFiles."template/cpp/CompetitiveProgramming.tpl".source =
      ./template/cpp/CompetitiveProgramming.tpl;
    # The template of LaTeX
    extraFiles."template/tex/Basic.tpl".source = ./template/tex/Basic.tpl;
    extraFiles."template/tex/CSIE.tpl".source = ./template/tex/CSIE.tpl;
    # The template for Typst
    extraFiles."template/typ/CSIE.tpl".source = ./template/typst/CSIE.tpl;
  };
}
