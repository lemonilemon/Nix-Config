    local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
    if not vim.loop.fs_stat(lazypath) then
      vim.fn.system({
        "git",
        "clone",
        "--filter=blob:none",
        "https://github.com/folke/lazy.nvim.git",
        "--branch=stable", -- latest stable release
        lazypath,
      })
    end
    vim.opt.rtp:prepend(lazypath)

    vim.cmd([[
        set ai 
        " ignore case
        set ic
        set enc=utf8 
        set mouse=a 
        set autochdir 
        set autowrite 
        set nobackup
        "" delays
        set timeout ttimeout " separte mapping and keycode time
        set timeoutlen=500 " 500ms
        set ttimeoutlen=20 " 20ms
        "" files
        set noswapfile
        set undofile
        set undodir=~/.config/nvim/undodir
        "" search
        set incsearch
        set nohlsearch
        "" Sound
        set visualbell
        "" Style 
        syntax enable
        set background=dark
        hi Normal guibg=None ctermbg=None
        hi NonText guibg=None ctermbg=None
        set termguicolors
        hi CursorLine cterm=underline
        "" Indentation 
        set expandtab
        set tabstop=4
        set softtabstop=-1
        set shiftwidth=4
        set smartindent
        "" Line number 
        " add line number
        set number 
        " columns used for the line number
        set numberwidth=4
        augroup numbertoggle
            autocmd! 
            autocmd BufEnter,FocusGained,InsertLeave,WinEnter * if &nu && mode() != "i" | set rnu | endif 
            autocmd BufLeave,FocusLost,InsertEnter,WinLeave * if &nu | set nornu | endif
        augroup END
        "" Clipboard
        set clipboard+=unnamedplus
        let g:clipboard = {
        \	'name': 'wsl-clip',
        \	'copy': {
        \		'+': 'clip.exe',
        \		'*': 'clip.exe',
        \	},
        \	'paste': {
        \      '+': 'powershell.exe -c [Console]::Out.Write($(Get-Clipboard -Raw).tostring().replace("`r", ""))',
        \      '*': 'powershell.exe -c [Console]::Out.Write($(Get-Clipboard -Raw).tostring().replace("`r", ""))',
        \   },
        \	'cache_enabled': 0,
        \}
    ]])

    local plugins = {
        {
            "folke/which-key.nvim",
            event = "VeryLazy",
            lazy = true,
            init = function()
                vim.o.timeout = true
                vim.o.timeoutlen = 300
                vim.g.mapleader = " "
                vim.g.maplocalleader = " "
                local map = vim.api.nvim_set_keymap
                local opt = { noremap = true, silent = true }
                map('i', 'jj', '<ESC>', opt)
                map('n', '<F5>', ':NvimTreeToggle<CR>', opt)
                map('n', 'tt', ':NvimTreeFocus<CR>', opt)
                map('n', '<F6>', ':RunCode<CR>', opt)
                vim.keymap.set('n', '<F7>', function()
                    vim.fn.feedkeys(':Template ')
                end, { remap = true})
            end,
            opts = {
                -- your configuration comes here
                -- or leave it empty to use the default settings
                -- refer to the configuration section below
            },
        },

        {
            "xiyaowong/nvim-transparent",
            lazy = true,
            opts = {},
            init = function()
                vim.g.transparent_enabled = true
            end,
        },

        {
            "catppuccin/nvim",
            name = "catppuccin",
            priority = 1000,
            opts = {
                flavour = "mocha",
                transparent_background = true,
            },
            config = function()
                vim.cmd.colorscheme "catppuccin"
            end,
        },

        {
            "neovim/nvim-lspconfig",
            event = { "BufReadPre", "BufNewFile" },
            dependencies = {
                { "folke/neoconf.nvim", cmd = "Neoconf", config = true },
                { "folke/neodev.nvim", opts = { experimental = { pathStrict = true } } },
            },
            opts = {
                servers = {
                    lua_ls = {
                        settings = {
                            Lua = {
                                diagnostics = {
                                    -- Get the language server to recognize the `vim` global
                                    globals = {'vim'},
                                },
                            },
                        },
                    },--]]
                    pyright = {},
                    ccls = {
                        init_options = {
                            clang = {
                                extraArgs = { "-Wno-c++17-extensions" } ;
                            };
                        }
                    },
                    bashls = {},
                    cmake = {},
                    marksman = {},
                    ltex = {},
                    nil_ls = {},
                    vimls = {},
                },
            },
            -- you can do any additional lsp server setup here
            -- return true if you don't want this server to be setup with lspconfig
            -- @type table<string, fun(server:string, opts:_.lspconfig.options):boolean?>
            setup = {

            },
            config = function(_, opts)
                local servers = opts.servers
                for server, server_opts in pairs(servers) do
                    require("lspconfig")[server].setup(server_opts)
                end
            end,
        },

    {
        "nvim-tree/nvim-tree.lua",
        dependencies = { "nvim-tree/nvim-web-devicons" },
        opts = {
            sort_by = "case_sensitive",
            view = {
                width = 40,
            },
            renderer = {
              group_empty = true,
            },
            filters = {
                dotfiles = true,
            },
        },
        config = function(_, opts)
            require("nvim-tree").setup(opts)
        end,
    },


    {
        "nvim-lualine/lualine.nvim",
        opts = {},
    },
}

local opts = {
    install = {
        missing = true,
        colorscheme = { "catppuccin" }
    },
}

require("lazy").setup(plugins, opts)
