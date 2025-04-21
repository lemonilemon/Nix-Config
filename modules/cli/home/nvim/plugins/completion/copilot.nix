{
  programs.nixvim = {
    plugins = {
      copilot-cmp = {
        enable = true;
      };
      copilot-lua = {
        enable = true;
        settings = {
          suggestion = {
            enabled = false;
          };
          panel = {
            enabled = false;
          };
          filetypes = {
            markdown = true;
            help = true;
          };
        };
      };
      copilot-chat = {
        enable = true;
        settings = {
          auto_insert_mode = true;
          show_help = true;
          question_header = "  User ";
          answer_header = "  Copilot ";
          model = "gemini-2.5-pro"; # Use the latest model available
        };
      };
      # For copilot chat
      edgy.settings = {
        right = [
          {
            ft = "copilot-chat";
            title = "Copilot Chat";
          }
        ];
      };
    };
  };
}
