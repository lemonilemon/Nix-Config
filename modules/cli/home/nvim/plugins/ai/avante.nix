{
  programs.nixvim.plugins = {
    avante = {
      enable = true;
      settings = {
        diff = {
          autojump = true;
          debug = false;
          list_opener = "copen";
        };
        highlights = {
          diff = {
            current = "DiffText";
            incoming = "DiffAdd";
          };
        };
        hints = {
          enabled = true;
        };
        mappings = {
          diff = {
            both = "cb";
            next = "]x";
            none = "c0";
            ours = "co";
            prev = "[x";
            theirs = "ct";
          };
        };
        provider = "copilot";
        auto_suggestions_provider = "copilot";
        providers = {
          copilot = {
            endpoint = "https://api.githubcopilot.com";
          };
        };

        vendors = {
          copilot_claude = {
            __inherited_from = "copilot";
            model = "claude-sonnet-4";
            display_name = "copilot/claude-4";
          };
          copilot_gemini = {
            __inherited_from = "copilot";
            model = "gemini-2.5-pro-preview-06-05";
            display_name = "copilot/gemini-2.5-pro";
          };
          copilot_o1 = {
            __inherited_from = "copilot";
            model = "o1";
            display_name = "copilot/chatgpt-o1";
          };
          copilot_o3_mini = {
            __inherited_from = "copilot";
            model = "o3-mini";
            display_name = "copilot/chatgpt-o3-mini";
          };
          copilot_o4_mini = {
            __inherited_from = "copilot";
            model = "o4-mini";
            display_name = "copilot/chatgpt-o4-mini";
          };
        };
        windows = {
          sidebar_header = {
            align = "center";
            rounded = true;
          };
          width = 35;
          wrap = true;
        };
      };
    };
    edgy.settings = {
      right = [
        {
          ft = "avante";
          title = "Avante";
        }
      ];
    };
  };
}
