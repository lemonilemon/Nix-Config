local nvim_lsp = require("lspconfig")
local servers = {
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
}

for lsp, opt in pairs(servers) do
    nvim_lsp[lsp].setup(opt)
end

require("lemonilemon.lspconfig.keymaps")
