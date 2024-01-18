return {
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
}

