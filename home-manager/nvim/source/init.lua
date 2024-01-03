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
            map('n', '<F7>', ":Template ", { noremap = true })
        end,
        config = true,
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
        "neovim/nvim-lspconfig",
        event = { "BufReadPre", "BufNewFile" },
        dependencies = {
            {
                "folke/neodev.nvim",
                opts = {
                    experimental = { pathStrict = true },
                    override = function(root_dir, library)
                        if root_dir:find("~/nixos-config/home-manager/nvim/source", 1, true) == 1 then
                            library.enabled = true
                            library.plugins = true
                        end
                    end,
                },
            },
        },
        opts = {
            servers = {
                lua_ls = {
                    settings = {
                        Lua = {
                            completion = {
                                callSnippet = "Replace"
                            },
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
            -- you can do any additional lsp server setup here
            -- return true if you don't want this server to be setup with lspconfig
            -- @type table<string, fun(server:string, opts:_.lspconfig.options):boolean?>
            setup = {

            },
        },
        config = function(_, opts)
            local capabilities = require('cmp_nvim_lsp').default_capabilities()
            local servers = opts.servers
            local function setup(server)
                local server_opts = vim.tbl_deep_extend("force", {
                    capabilities = vim.deepcopy(capabilities),
                }, servers[server] or {})
                if opts.setup[server] then
                    if opts.setup[server](server, server_opts) then
                        return
                    end
                elseif opts.setup["*"] then
                    if opts.setup["*"](server, server_opts) then
                        return
                    end
                end
                require("lspconfig")[server].setup(server_opts)
            end

            for server, _ in pairs(servers) do
                setup(server)
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
                theme = "auto"
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
            map('n', '<leader>ft', ":Telescope find_template<CR>", mapopt)
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
            local ls = require "luasnip"
            vim.keymap.set({"i"}, "<C-K>", function() ls.expand() end, {silent = true})
            vim.keymap.set({"i", "s"}, "<C-L>", function() ls.jump( 1) end, {silent = true})
            vim.keymap.set({"i", "s"}, "<C-J>", function() ls.jump(-1) end, {silent = true})
            vim.keymap.set({"i", "s"}, "<C-E>", function()
                if ls.choice_active() then
                ls.change_choice(1)
                    end
            end, {silent = true})
        end,
    },

    {
        "hrsh7th/nvim-cmp",
        event = { "BufReadPre", "BufNewFile" },
        dependencies = {
            "neovim/nvim-lspconfig",
            "hrsh7th/cmp-nvim-lsp",
            "hrsh7th/cmp-buffer",
            "hrsh7th/cmp-path",
            "hrsh7th/cmp-cmdline",
            "L3MON4D3/LuaSnip",
            "saadparwaiz1/cmp_luasnip",
        },
        opts = function()
            local has_words_before = function()
                unpack = unpack or table.unpack
                local line, col = unpack(vim.api.nvim_win_get_cursor(0))
                return col ~= 0 and vim.api.nvim_buf_get_lines(0, line - 1, line, true)[1]:sub(col, col):match("%s") == nil
            end
            local luasnip = require("luasnip")
            local cmp = require("cmp")
            return {
                snippet = {
                    expand = function(args)
                        require('luasnip').lsp_expand(args.body)
                    end,
                },
                sources = cmp.config.sources(
                    {
                        { name = 'nvim_lsp' },
                        { name = 'luasnip' }, -- For luasnip users.
                    },

                    {
                        { name = "buffer" },
                        { name = "path" },
                    }
                ),
                mapping = {
                    ["<Tab>"] = cmp.mapping(function(fallback)
                      if cmp.visible() then
                        cmp.select_next_item()
                      -- You could replace the expand_or_jumpable() calls with expand_or_locally_jumpable() 
                      -- that way you will only jump inside the snippet region
                      elseif luasnip.expand_or_jumpable() then
                        luasnip.expand_or_jump()
                      elseif has_words_before() then
                        cmp.complete()
                      else
                        fallback()
                      end
                    end, { "i", "s" }),

                    ["<S-Tab>"] = cmp.mapping(function(fallback)
                      if cmp.visible() then
                        cmp.select_prev_item()
                      elseif luasnip.jumpable(-1) then
                        luasnip.jump(-1)
                      else
                        fallback()
                      end
                    end, { "i", "s" }),
                },
            }
        end,
    },

}

local opts = {
    checker = {
        enabled = true,
    },
    install = {
        missing = true,
        colorscheme = { "catppuccin" }
    },
    defaults = {
        lazy = false,
        version = false,
    },
}

require("lazy").setup(plugins, opts)
