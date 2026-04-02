{
  programs.nixvim = {
    plugins.hardtime = {
      enable = true;
      settings = {
        disabled_filetypes = [
          "neo-tree"
          "neo-tree-popup"
          "neo-tree-preview"
          "qf"
          "help"
          "lazy"
          "mason"
          "notify"
          "toggleterm"
          "lazyterm"
        ];
      };
    };
  };
}
