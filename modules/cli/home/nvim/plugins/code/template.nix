{
  pkgs,
  inputs,
  ...
}:
{
  programs.nixvim = {
    extraPlugins = with pkgs.vimUtils; [
      (buildVimPlugin {
        pname = "template.nvim";
        version = "2025-11-14";
        src = inputs.template-nvim;
      })
    ];
    extraConfigLua = ''
      require("telescope").load_extension('find_template')

      require("template").setup({
        temp_dir = "~/.config/nvim/template",
        author = "lemonilemon",
      })
    '';
    extraFiles."template/c/Basic.tpl".source = ./template/c/Basic.tpl;
    extraFiles."template/cpp/Basic.tpl".source = ./template/cpp/Basic.tpl;
    extraFiles."template/cpp/CompetitiveProgramming.tpl".source =
      ./template/cpp/CompetitiveProgramming.tpl;
    extraFiles."template/cpp/CP_Atcoder.tpl".source = ./template/cpp/CP_Atcoder.tpl;
    extraFiles."template/tex/Basic.tpl".source = ./template/tex/Basic.tpl;
    extraFiles."template/tex/CSIE.tpl".source = ./template/tex/CSIE.tpl;
    extraFiles."template/typ/CSIE.tpl".source = ./template/typst/CSIE.tpl;
    extraFiles."template/nix/Project.tpl".source = ./template/nix/Project.tpl;
  };
}
