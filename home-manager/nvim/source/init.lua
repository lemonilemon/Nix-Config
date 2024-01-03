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

    local map = vim.api.nvim_set_keymap
    local mapopt = { noremap = true, silent = true }


    local plugins = {
        {
            "folke/which-key.nvim",
            event = "VeryLazy",
            -- lazy = true,
            init = function()
                vim.o.timeout = true
                vim.o.timeoutlen = 300
                vim.g.mapleader = " "
                vim.g.maplocalleader = " "
                map('i', 'jj', '<ESC>', mapopt)
                map("n", " ", "<Nop>", mapopt)
                map('n', '<F6>', ':RunCode<CR>', mapopt)
                map('n', '<F7>', ":Template ", mapopt)
            end,
            opts = {
                -- your configuration comes here
                -- or leave it empty to use the default settings
                -- refer to the configuration section below
            },
        },

        {
            "xiyaowong/nvim-transparent",
            opts = {
                extra_groups = {
                    "NormalFloat", -- for floats
                    "NvimTreeNormal", -- NvimTree
                    "NvimTreeNormalNC",
                    "NvimTreeWinSeparator",
                    "NvimTreeCursorLine",
                    "NvimTreeCursorColumn",
                    "TelescopeNormal", -- Telescope
                    "TelescopeBorder",
                    "LspFloatWinNormal", -- Lsp
                    "LspFloatWinBorder",
                },
            },
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
        lazy = true,
        dependencies = { "nvim-tree/nvim-web-devicons" },
        cmd = {
            "NvimTreeToggle",
            "NvimTreeFocus",
        },
        init = function()
            vim.opt.termguicolors = true
            map('n', 'tt', ':NvimTreeFocus<CR>', mapopt)
        end,
        opts = {
            sort_by = "case_sensitive",
            view = {
                width = 40,
            },
            renderer = {
                group_empty = true,
            },
            filters = {
                dotfiles = false,
            },
        },
    },


    {
        "nvim-lualine/lualine.nvim",
        opts = {
            options = {
                icons_enabled = true,
                theme  = "auto"
            },
        },
    },

    {
        "nvim-telescope/telescope.nvim", tag = "0.1.5",
        dependencies = {
            "nvim-lua/plenary.nvim",
            {
                "nvim-telescope/telescope-fzf-native.nvim",
                build = "make",
            }
        },
        init = function()
            map('n', '<leader>ff', ":Telescope find_files<CR>", mapopt)
            map('n', '<leader>fg', ":Telescope live_grep<CR>", mapopt)
            map('n', '<leader>fb', ":Telescope buffers<CR>", mapopt)
            map('n', '<leader>fh', ":Telescope help_tags<CR>", mapopt)
        end,
        opts = {
            defaults = {
                layout_config = {
                    horizontal = {
                        preview_cutoff = 0,
                    },
                },
            },
            extensions = {
                fzf = {
                    fuzzy = true,                    -- false will only do exact matching
                    override_generic_sorter = true,  -- override the generic sorter
                    override_file_sorter = true,     -- override the file sorter
                    case_mode = "smart_case",        -- or "ignore_case" or "respect_case"
                },
            },
        },
    },

    {
        "nvim-treesitter/nvim-treesitter",
        main = "nvim-treesitter.configs",
        opts = {
            ensure_installed = {
              "c",
              "cpp",
              "lua",
              "vim",
              "vimdoc",
              "query",
              "python",
            },
            sync_install = true,
            auto_install = true,
            ignore_install = {},
            highlight = {
                enable = true,
                disable = {
                    "latex", -- vimtex
                },
                additional_vim_regex_highlighting = { "markdown", "latex" },
            },
        },
        build = ":TSUpdate",
    },
    {
        "CRAG666/code_runner.nvim",
        lazy = true,
        cmd = "RunCode",
        opts = {
            mode = "term",
            focus = true,
            startinsert = true,
            before_run_filetype = function()
                vim.cmd(":w")
            end,
            filetype = {
                java = {
                    "cd $dir &&",
                    "javac $fileName &&",
                    "java $fileNameWithoutExt"
                },
                python = "python3 -u",
                c = {
                    "cd $dir &&",
                    "gcc $fileName -o $fileNameWithoutExt &&",
                    "$dir/$fileNameWithoutExt &&",
                    "rm $dir/$fileNameWithoutExt",
                },
                cpp = {
                    "cd $dir &&",
                    "g++ $fileName -o $fileNameWithoutExt &&",
                    "$dir/$fileNameWithoutExt &&",
                    "rm $dir/$fileNameWithoutExt",
                },
                typescript = "deno run",
                rust = {
                    "cd $dir &&",
                    "rustc $fileName &&",
                    "$dir/$fileNameWithoutExt"
                },
                tex = function()
                    vim.api.nvim_command("VimtexCompile")
                end,
                markdown = function()
                    require("code_runner.hooks.preview_pdf").run {
                        command = "pandoc",
                        args = { "$fileName", "-o", "$tmpFile", "-t", "pdf" },
                        preview_cmd = "wsl-open > /dev/null",
                        overwrite_output = "/tmp",
                    }
                end,
            },
        },
    },

    {
        "nvimdev/template.nvim",
        event = "VeryLazy",
        init = function()
            require("telescope").load_extension('find_template')
        end,
        opts = {
            temp_dir = "~/.config/nvim/template",
            author = "lemonilemon",
            email = "imlemonilemon@gmail.com",
        },
    },

    {
        "lervag/vimtex",
        lazy = true,
        ft = "tex",
        init = function()
            vim.g.vimtex_view_method = "zathura"
            vim.g.vimtex_view_general_viewer = "zathura"
            vim.g.vimtex_compiler_latexmk_engines = {
                _ = "-pdflatex"
            }
            vim.g.tex_comment_nospell = 1
            vim.g.vimtex_compiler_progname = "nvr"
            vim.g.vimtex_view_general_options = [[--unique file:@pdf\#src:@line@tex]]
        end,
    },

    {
        "L3MON4D3/LuaSnip",
        build = "make install_jsregexp",
        init = function()
        end,
    }
}

local opts = {
    install = {
        missing = true,
        colorscheme = { "catppuccin" }
    },
}

require("lazy").setup(plugins, opts)