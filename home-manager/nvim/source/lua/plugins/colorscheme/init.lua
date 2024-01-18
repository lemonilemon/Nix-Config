local files = {
    -- put your plugins here
    "catppuccin",
    "transparent",
};

local ret = {};

-- load plugins
for _, file in ipairs(files) do
    table.insert(ret, require("plugins.colorscheme." .. file));
end;

return ret;
