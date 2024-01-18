local files = {
    -- put your files here
    "vimtex",
    "neorg",
};

local ret = {
    -- put your plugins here
};

-- load plugins
for _, file in ipairs(files) do
    table.insert(ret, require("plugins.files." .. file));
end;

return ret;
