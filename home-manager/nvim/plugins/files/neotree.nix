{
  programs.nixvim.plugins = {
    # https://nix-community.github.io/nixvim/plugins/nvim-tree/index.html
    neo-tree = {
      enable = true;
      enableDiagnostics = true;
      enableGitStatus = true;
      enableModifiedMarkers = true;
      enableRefreshOnWrite = true;
      closeIfLastWindow = true;
    };
  };
}
