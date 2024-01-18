local files = {
    -- put your files here
    "copilot",
    "lsp",
    "treesitter",
    "cmp",
    "comment",
    "coderunner",
    "template",
};

local ret = {
    -- put your plugins here
};

-- load plugins
for _, file in ipairs(files) do
    table.insert(ret, require("plugins.code." .. file));
end;

return ret;

