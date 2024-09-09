{
  programs.nixvim = {
    enable = true;
    plugins = {
      # https://nix-community.github.io/nixvim/plugins/comment/index.html
      comment = {
        enable = true;
      };
    };
  };
}
