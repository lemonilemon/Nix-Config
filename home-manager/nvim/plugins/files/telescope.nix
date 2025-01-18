{
  programs.nixvim.plugins = {
    # https://nix-community.github.io/nixvim/plugins/telescope/index.html
    telescope = {
      enable = true;
      extensions = {
        fzf-native = {
          enable = true;
          settings = {
            fuzzy = true;
            override_generic_sorter = true;
            override_file_sorter = true;
            case_mode = "smart_case";
          };
        };
      };
    };
  };
}
