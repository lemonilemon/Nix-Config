return {
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
}

