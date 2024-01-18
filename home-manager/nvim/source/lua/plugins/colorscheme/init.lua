local files = {
    -- put your files here
    "catppuccin",
    "transparent",
};

local ret = {
    -- put your plugins here
};

-- load plugins
for _, file in ipairs(files) do
    table.insert(ret, require("plugins.colorscheme." .. file));
end;

return ret;
