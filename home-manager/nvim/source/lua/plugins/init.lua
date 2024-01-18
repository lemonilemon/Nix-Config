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
}

local plugdirs = {
    "colorscheme",
    "code",
};

for _, plugdir in ipairs(plugdirs) do
    for _, plug in ipairs(require("plugins." .. plugdir)) do
        table.insert(plugins, plug);
    end;
end;

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
