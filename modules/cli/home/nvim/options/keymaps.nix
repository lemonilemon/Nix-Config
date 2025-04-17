{
  programs.nixvim = {
    globals = {
      mapleader = " ";
      maplocalleader = " ";
    };
    keymaps = [
      {
        mode = "n";
        key = " ";
        action = "<Nop>";
        options = {
          desc = "Map Space to <Nop> to make it the leader key only";
          noremap = true;
          silent = true;
        };
      }

      # Move to window using the <ctrl> hjkl keys (Use zellij-nav instead)
      # {
      #   mode = "n";
      #   key = "<C-h>";
      #   action = "<C-w>h";
      #   options = {
      #     desc = "Go to Left Window";
      #   };
      # }
      #
      # {
      #   mode = "n";
      #   key = "<C-j>";
      #   action = "<C-w>j";
      #   options = {
      #     desc = "Go to Lower Window";
      #   };
      # }
      # {
      #   mode = "n";
      #   key = "<C-k>";
      #   action = "<C-w>k";
      #   options = {
      #     desc = "Go to Upper Window";
      #   };
      # }
      # {
      #   mode = "n";
      #   key = "<C-l>";
      #   action = "<C-w>l";
      #   options = {
      #     desc = "Go to Right Window";
      #   };
      # }
      # Use `H` and `L` to move between buffers
      {
        mode = "n";
        key = "<S-h>";
        action = "<cmd>bprevious<CR>";
        options = {
          desc = "Go to Previous Buffer";
          noremap = true;
          silent = true;
        };
      }
      {
        mode = "n";
        key = "<S-l>";
        action = "<cmd>bnext<CR>";
        options = {
          desc = "Go to Next Buffer";
          noremap = true;
          silent = true;
        };
      }
      # Copilot-chat keymaps
      {
        mode = [
          "n"
          "v"
        ];
        key = "<leader>aa";
        action = "<cmd>CopilotChatOpen<CR>";
        options = {
          desc = "Open Copilot Chat";
          noremap = true;
          silent = true;
        };
      }
      {
        mode = [
          "n"
          "v"
        ];
        key = "<leader>ax";
        action = "<cmd>CopilotChatReset<CR>";
        options = {
          desc = "Reset Copilot Chat";
          noremap = true;
          silent = true;
        };
      }

      {
        mode = "i";
        key = "jj";
        action = "<ESC>";
        options = {
          desc = "Map jj to Escape";
          noremap = true;
          silent = true;
        };
      }

      # Code Runner
      {
        mode = "n";
        key = "<F6>";
        action = "<cmd>RunCode<CR>";
        options = {
          desc = "Run code with <F6>";
          noremap = true;
          silent = true;
        };
      }

      # Template
      {
        mode = "n";
        key = "<F7>";
        action = ":Template ";
        options = {
          desc = "Insert template with <F7>";
          noremap = true;
        };
      }

      # LazyGit
      {
        mode = "n";
        key = "<F8>";
        action = "<cmd>LazyGit<CR>";
        options = {
          desc = "Open LazyGit with <F8>";
          noremap = true;
          silent = true;
        };
      }

      # neo-tree
      {
        mode = "n";
        key = "<leader>fe";
        action = "<cmd>Neotree<CR>";
        options = {
          desc = "Open Neotree with <leader>fe";
          noremap = true;
          silent = true;
        };
      }

      # Telescope
      {
        mode = "n";
        key = "<leader>ff";
        action = "<cmd>Telescope find_files<CR>";
        options = {
          desc = "Find files in Telescope with <leader>ff";
          noremap = true;
          silent = true;
        };
      }

      {
        mode = "n";
        key = "<leader>fg";
        action = "<cmd>Telescope find_files<CR>";
        options = {
          desc = "Grep in Telescope with <leader>fg";
          noremap = true;
          silent = true;
        };
      }

      {
        mode = "n";
        key = "<leader>fb";
        action = "<cmd>Telescope buffers<CR>";
        options = {
          desc = "Find buffers in Telescope with <leader>fb";
          noremap = true;
          silent = true;
        };
      }

      {
        mode = "n";
        key = "<leader>fh";
        action = "<cmd>Telescope help_tags<CR>";
        options = {
          desc = "Find helps in Telescope with <leader>fh";
          noremap = true;
          silent = true;
        };
      }

      {
        mode = "n";
        key = "<leader>ft";
        action = "<cmd>Telescope find_template<CR>";
        options = {
          desc = "Find template (Template.nvim) in Telescope with <leader>ft";
          noremap = true;
          silent = true;
        };
      }

      # Noice
      {
        mode = "n";
        key = "<leader>nd";
        action = "<cmd>NoiceDismiss<CR>";
        options = {
          desc = "Dismiss Noice with <leader>nd";
          noremap = true;
          silent = true;
        };
      }

    ];
  };
}
