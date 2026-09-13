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
      auto-reload = {
        clear = true;
      };
    };

    # [[ AutoCommands ]]
    # https://nix-community.github.io/nixvim/NeovimOptions/autoCmd/index.html
    autoCmd = [
      {
        event = [
          "FocusGained"
          "BufEnter"
          "CursorHold"
          "CursorHoldI"
        ];
        group = "auto-reload";
        desc = "Reload buffer when file changes externally (e.g. AI edits)";
        command = "checktime";
      }
      {
        event = [ "InsertEnter" ];
        desc = "Toggle between relative and absolute line numbers when entering insert mode";
        callback.__raw = ''
          function()
              vim.opt.relativenumber = false;
          end
        '';
      }
      {
        event = [ "InsertLeave" ];
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

      encoding = "utf-8";
      fileencoding = "utf-8";

      autochdir = true;
      autowrite = true;
      autoread = true;

      # fire CursorHold sooner for faster auto-reload detection (community standard)
      updatetime = 300;

      ttimeout = true; # separate mapping and keycode timeout
      timeoutlen = 300;
      ttimeoutlen = 20;

      backup = false;
      swapfile = false;
      undofile = true;

      number = true;
      numberwidth = 4;

      incsearch = true;
      hlsearch = false;

      visualbell = true;

      background = "dark";
      termguicolors = true;

      expandtab = true;
      tabstop = 4;
      softtabstop = -1;
      shiftwidth = 4;

      showmode = false;

      # Preview substitutions live as you type
      inccommand = "split";
    };
    clipboard = {
      providers = {
        wl-copy.enable = true;
      };
      register = "unnamedplus";
    };
  };
}
