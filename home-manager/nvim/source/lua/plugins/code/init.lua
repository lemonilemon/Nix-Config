local files = {
    -- put your plugins here
    "copilot",
    "lsp",
    "treesitter",
    "cmp",
    "comment",
    "coderunner",
    "template",
};

local ret = {};

-- load plugins
for _, file in ipairs(files) do
    table.insert(ret, require("plugins.code." .. file));
end;

return ret;

