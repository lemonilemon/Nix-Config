local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
  vim.fn.system({
    "git",
    "clone",
    "--filter=blob:none",
    "https://github.com/folke/lazy.nvim.git",
    "--branch=stable", -- latest stable release
    lazypath,
  })
end

vim.opt.rtp:prepend(lazypath)


local plugins = {};

local plugdirs = {
    "colorscheme",
    "code",
    "files",
    "ui",
    "utils",
};

for _, plugdir in ipairs(plugdirs) do
    for _, plug in ipairs(require("plugins." .. plugdir)) do
        table.insert(plugins, plug);
    end;
end;

local opts = {
    checker = {
        enabled = true,
    },
    install = {
        missing = true,
        colorscheme = { "catppuccin" }
    },
    defaults = {
        lazy = false,
        version = false,
    },
}

require("lazy").setup(plugins, opts)
