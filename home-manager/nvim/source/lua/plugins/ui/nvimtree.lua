return {
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
}
