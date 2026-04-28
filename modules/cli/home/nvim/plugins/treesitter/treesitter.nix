{ pkgs, ... }:
{
  # https://nix-community.github.io/nixvim/plugins/treesitter/index.html
  home.packages = with pkgs; [
    tree-sitter
  ];
  programs.nixvim = {
    plugins.treesitter = {
      enable = true;
      indent.enable = true;
      highlight = {
        enable = true;
        disable = [ "latex" ];
      };
      folding.enable = false;
      nixvimInjections = true;
    };
  };
}
