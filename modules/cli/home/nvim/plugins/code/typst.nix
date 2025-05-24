{
  lib,
  pkgs,
  ...
}:
{
  home.packages = with pkgs; [
    typst
  ];
  programs.nixvim = {
    plugins = {
      typst-preview = {
        enable = true;
      };
    };
  };
}
