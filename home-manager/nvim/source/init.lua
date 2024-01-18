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

require("options.base");
require("options.keymaps");

-- mappings
local plugins = {
    {
        "dstein64/vim-startuptime",
        cmd = "StartupTime",
        config = function()
            vim.g.startuptime_tries = 10
        end,
    },

    {
        "folke/which-key.nvim",
        event = "VeryLazy",
        -- lazy = true,
        config = {},
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
                "DiagnosticVirtualTextError", -- Diagnostic
                "DiagnosticVirtualTextWarn",
                "DiagnosticVirtualTextInfo",
                "DignosticVirtualTextHint",
            },
        },
        init = function()
            vim.g.transparent_enabled = true
        end,
    },

    {
        "folke/noice.nvim",
        event = "VeryLazy",
        opts = {
            -- this is suggested setup
            lsp = {
                override = {
                    ["vim.lsp.util.convert_input_to_markdown_lines"] = true,
                    ["vim.lsp.util.stylize_markdown"] = true,
                    ["cmp.entry.get_documentation"] = true,
                },
            },
            presets = {
                bottom_search = true, -- use a classic bottom cmdline for search
                command_palette = true, -- position the cmdline and popupmenu together
                long_message_to_split = true, -- long messages will be sent to a split
                inc_rename = false, -- enables an input dialog for inc-rename.nvim
                lsp_doc_border = false, -- add a border to hover docs and signature help
            },
        },
        dependencies = {
            "MunifTanjim/nui.nvim",
            {
                "rcarriga/nvim-notify",
                opts = {
                    background_colour = "#000000",
                },
            },
        },
    },

    {
        "neovim/nvim-lspconfig",
        event = { "BufReadPre", "BufNewFile" },
        dependencies = {
            {
                "folke/trouble.nvim",
                event = "VeryLazy",
                dependencies = { "nvim-tree/nvim-web-devicons" },
                opts = {
                    auto_close = true, -- automatically close the list when you have no diagnostics
                    auto_fold = false,
                    signs = {
                        error = "",
                        warning = "",
                        hint = "",
                        information = "",
                        other = "",
                    },
                },
                use_diagnostic_signs = false,
            },

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
            diagnostics = {
                underline = true,
                update_in_insert = true,
                virtual_text = {
                    spacing = 4,
                    source = "if_many",
                    prefix = "icons",
                },
                severity_sort = true,
            },
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
                tsserver = {},
                bashls = {},
                cmake = {},
                marksman = {},
                ltex = {},
                nil_ls = {},
                vimls = {},
                rust_analyzer = {
                    settings = {
                        ["rust-analyzer"] = {},
                    },
                },
                taplo = {},
                cssls = {},
                html = {},
                jsonls = {},
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
        dependencies = { "nvim-tree/nvim-web-devicons" },
        cmd = {
            "NvimTreeToggle",
            "NvimTreeFocus",
        },
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
        dependencies = {
            { "AndreM222/copilot-lualine" },
        },
        opts = {
            options = {
                icons_enabled = true,
                theme = "auto",
                component_separators = { left = '', right = '' },
                section_separators = { left = '', right = '' },
                disabled_filetypes = {},
                always_divide_middle = true
            },
            sections = {
                lualine_a = { 'mode' },
                lualine_b = { 'branch', 'diff',
                    {
                        'diagnostics',
                        sources = { "nvim_diagnostic" },
                        symbols = { error = ' ', warn = ' ', info = ' ', hint = ' ' }
                    }
                },
                lualine_c = { 'filename' },
                lualine_x = { 'copilot' ,'encoding', 'fileformat', 'filetype' }, -- I added copilot here
                lualine_y = { 'progress' },
                lualine_z = { 'location' }
            },
            inactive_sections = {
                lualine_a = {},
                lualine_b = {},
                lualine_c = { 'filename' },
                lualine_x = { 'location' },
                lualine_y = {},
                lualine_z = {}
            },
            tabline = {},
            extensions = {}
        },
    },

    {
        "kdheepak/lazygit.nvim",
        dependencies = {
            "nvim-lua/plenary.nvim",
        },
        cmd = {
            "LazyGit",
            "LazyGitConfig",
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
            indent = {
                enable = true,
            },
        },
        build = ":TSUpdate",
    },
    {
        "CRAG666/code_runner.nvim",
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
        cmd = { "Template", "TemProject" },
        init = function()
            require("telescope").load_extension('find_template')
        end,
        opts = {
            temp_dir = "~/.config/nvim/template",
            author = "lemonilemon",
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
        lazy = true,
        build = "make install_jsregexp",
        opts = {
            history = true,
            delete_check_events = "TextChanged",
        },
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
            local luasnip = require("luasnip")
            local cmp = require("cmp")
            return {
                snippet = {
                    expand = function(args)
                        luasnip.lsp_expand(args.body)
                    end,
                },
                sources = cmp.config.sources({
                    { name = "copilot" },
                    { name = "nvim_lsp" },
                    { name = "luasnip" }, -- For luasnip users.
                    { name = "buffer" },
                    { name = "path" },
                }),
                experimental = {
                    ghost_text = {
                        hl_group = "CmpGhostText",
                    },
                },
                mapping = cmp.mapping.preset.insert({
                    ["<C-j>"] = cmp.mapping.select_next_item({ behavior = cmp.SelectBehavior.Insert }),
                    ["<C-k>"] = cmp.mapping.select_prev_item({ behavior = cmp.SelectBehavior.Insert }),
                    ['<C-b>'] = cmp.mapping.scroll_docs(-4),
                    ['<C-f>'] = cmp.mapping.scroll_docs(4),
                    ['<C-Space>'] = cmp.mapping.complete(),
                    ['<C-CR>'] = cmp.mapping.abort(),
                    ['<CR>'] = cmp.mapping.confirm({ select = true }), -- Accept currently selected item. Set `select` to `false` to only confirm explicitly selected items.
                }),
            }
        end,
    },

    {
        "nvim-neorg/neorg",
        ft = "norg",
        build = ":Neorg sync-parsers",
        dependencies = { "nvim-lua/plenary.nvim" },
        opts = {
            load = {
                ["core.defaults"] = {}, -- Loads default behaviour
                ["core.concealer"] = {}, -- Adds pretty icons to your documents
                ["core.dirman"] = { -- Manages Neorg workspaces
                    config = {
                        workspaces = {
                            notes = "~/notes",
                        },
                    },
                },
            },
        },
    },

    -- codeium with some issues
    -- {
        --"Exafunction/codeium.nvim",
        --event = "VeryLazy",
        --dependencies = {
            --"nvim-lua/plenary.nvim",
            --"hrsh7th/nvim-cmp",
        --},
        --opts = {},
    --},

    {
        "zbirenbaum/copilot.lua",
        dependencies = {
            {
                "zbirenbaum/copilot-cmp",
                event = { "InsertEnter", "LspAttach" },
                opts = {},
            },
            { "AndreM222/copilot-lualine" },
        },
        build = ":Copilot auth",
        event = "VeryLazy",
        opts = {
            suggestion = { enabled = false },
            panel = { enabled = false },
            filetypes = {
                markdown = true,
                help = true,
            },
        },
    },

    {
        'numToStr/Comment.nvim',
        dependencies = {
            { "JoosepAlviste/nvim-ts-context-commentstring" },
        },
        event = "VeryLazy",
        opts = {},
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
