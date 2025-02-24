{
  programs.nixvim = {
    # [[ Globals ]]
    # https://nix-community.github.io/nixvim/NeovimOptions/index.html?highlight=globals#globals
    globals = { };

    # [[ AutoGroups ]]
    # https://nix-community.github.io/nixvim/NeovimOptions/autoGroups/index.html
    autoGroups = {
      number-toggle = {
        clear = true;
      };
    };

    # [[ AutoCommands ]]
    # https://nix-community.github.io/nixvim/NeovimOptions/autoCmd/index.html
    autoCmd = [
      {
        event = [
          "InsertEnter"
        ];
        desc = "Toggle between relative and absolute line numbers when entering insert mode";
        callback.__raw = ''
          function()
              vim.opt.relativenumber = false;
          end
        '';
      }
      {
        event = [
          "InsertLeave"
        ];
        desc = "Toggle between relative and absolute line numbers when leaving insert mode";
        callback.__raw = ''
          function()
              vim.opt.relativenumber = true;
          end
        '';
      }

      {
        event = "InsertEnter";
        desc = "Vertically center the document when entering insert mode";
        command = "norm zz";
      }
    ];

    # [[ Options ]]
    # https://nix-community.github.io/nixvim/NeovimOptions/index.html?highlight=globals#opts
    opts = {
      relativenumber = true;
      autoindent = true;
      ignorecase = true;
      mouse = "a";
      conceallevel = 1;

      # use utf-8 encoding
      encoding = "utf-8";
      fileencoding = "utf-8";

      autochdir = true;
      autowrite = true;

      # keymapping timeout
      ttimeout = true; # separate mapping and keycode timeout
      timeoutlen = 300;
      ttimeoutlen = 20;

      # use undofile
      backup = false;
      swapfile = false;
      undofile = true;

      # line number
      number = true;
      numberwidth = 4;

      # search
      incsearch = true;
      hlsearch = false;

      # sound
      visualbell = true;

      # style
      background = "dark";
      termguicolors = true;

      # indentation
      expandtab = true;
      tabstop = 4;
      softtabstop = -1;
      shiftwidth = 4;

      # Don't show mode
      showmode = false;

      # Preview substitutions live as you type
      inccommand = "split";
    };
    # Clipboard
    clipboard = {
      providers = {
        wl-copy.enable = true;
      };
      register = "unnamedplus";
    };
  };
}
