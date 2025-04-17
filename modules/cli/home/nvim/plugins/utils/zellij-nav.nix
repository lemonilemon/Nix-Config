{
  programs.nixvim = {
    plugins = {
      zellij-nav = {
        enable = true;
      };
    };
    keymaps = [
      # Move to different zellij pane using Alt + hjkl
      {
        mode = [
          "n"
          "v"
          "i"
        ];
        key = "<A-h>";
        action = "<cmd>ZellijNavigateLeft<cr>";
        options = {
          desc = "Go to Left Pane";
        };
      }
      {
        mode = [
          "n"
          "v"
          "i"
        ];
        key = "<A-j>";
        action = "<cmd>ZellijNavigateDown<cr>";
        options = {
          desc = "Go to Lower Pane";
        };
      }
      {
        mode = [
          "n"
          "v"
          "i"
        ];
        key = "<A-k>";
        action = "<cmd>ZellijNavigateUp<cr>";
        options = {
          desc = "Go to Upper Pane";
        };
      }
      {
        mode = [
          "n"
          "v"
          "i"
        ];
        key = "<A-l>";
        action = "<cmd>ZellijNavigateRight<cr>";
        options = {
          desc = "Go to Right Pane";
        };
      }

    ];
  };
}
