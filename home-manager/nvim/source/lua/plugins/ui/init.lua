local files = {
    -- put your files here
    "lualine",
    "noice",
    "nvimtree",
};

local ret = {
    -- put your plugins here
};

-- load plugins
for _, file in ipairs(files) do
    table.insert(ret, require("plugins.ui." .. file));
end;

return ret;
