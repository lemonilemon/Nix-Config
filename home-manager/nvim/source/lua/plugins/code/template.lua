return {
    "nvimdev/template.nvim",
    cmd = { "Template", "TemProject" },
    init = function()
        require("telescope").load_extension('find_template')
    end,
    opts = {
        temp_dir = "~/.config/nvim/template",
        author = "lemonilemon",
    },
}
