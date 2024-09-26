{
  programs.nixvim = {
    plugins = {
      # https://nix-community.github.io/nixvim/plugins/copilot-cmp/index.html
      copilot-cmp = {
        enable = true;
      };
      # https://nix-community.github.io/nixvim/plugins/copilot-lua/index.html
      copilot-lua = {
        enable = true;
        suggestion = {
          enabled = false;
        };
        panel = {
          enabled = false;
        };
      };
      # https://nix-community.github.io/nixvim/plugins/copilot-chat/index.html
      copilot-chat = {
        enable = true;
        settings = {
          auto_insert_mode = true;
          show_help = true;
          question_header = "  User ";
          answer_header = "  Copilot ";
        };
      };

    };
    extraConfigLua = ''
      require("copilot").setup({
          suggestion = { enabled = false },
          panel = { enabled = false },
          filetypes = {
              markdown = true;
              help = true;
          };
      })
    '';

  };
}
