{ pkgs, ... }:
{
  # https://nix-community.github.io/nixvim/plugins/treesitter/index.html
  programs.nixvim = {
    filetype.extension.liq = "liquidsoap";

    plugins.treesitter = {
      enable = true;

      settings = {
        indent = {
          enable = true;
        };
        highlight = {
          enable = true;
          disable = [
            "latex" # Use vimtex instead
          ];
        };
      };

      folding = false;
      languageRegister.liq = "liquidsoap";
      nixvimInjections = true;
      grammarPackages = pkgs.vimPlugins.nvim-treesitter.allGrammars;
    };

    extraConfigLua = ''
      local parser_config = require("nvim-treesitter.parsers").get_parser_configs()

      parser_config.liquidsoap = {
        filetype = "liquidsoap",
      }
    '';
  };
}
