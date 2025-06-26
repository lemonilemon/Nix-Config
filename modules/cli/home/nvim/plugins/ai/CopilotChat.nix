{
  programs.nixvim.plugins = {
    copilot-chat = {
      enable = true;
      settings = {
        auto_insert_mode = true;
        show_help = true;
        question_header = "  User ";
        answer_header = "  Copilot ";
        model = "claude-sonnet-4"; # Use the latest model available
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
}
