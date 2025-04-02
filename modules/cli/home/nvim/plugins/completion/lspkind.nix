{
  programs.nixvim.plugins = {
    # https://nix-community.github.io/nixvim/plugins/lspkind/index.html
    lspkind = {
      enable = true;
      symbolMap = {
        Copilot = "";
      };
      extraOptions = {
        maxwidth = 50;
        ellipsis_char = "...";
      };
    };
  };
}
