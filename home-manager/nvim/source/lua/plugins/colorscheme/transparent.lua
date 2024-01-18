return {
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
}
