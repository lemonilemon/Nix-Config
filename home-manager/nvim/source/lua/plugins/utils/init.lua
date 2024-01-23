
local files = {
    -- put your files here
    "telescope",
    "flash",
};

local ret = {
    -- put your plugins here
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
        config = {},
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

};

-- load plugins
for _, file in ipairs(files) do
    table.insert(ret, require("plugins.utils." .. file));
end;

return ret;

