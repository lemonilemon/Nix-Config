local opt = vim.opt;
local cmd = vim.cmd;
local fn = vim.fn;
local api = vim.api;
local augroup = api.nvim_create_augroup;
local autocmd = api.nvim_create_autocmd;

opt.ai = true;   -- auto indent
opt.ic = true;   -- ignore case
opt.mouse = "a"; -- enable mouse
opt.conceallevel = 1;
-- use utf-8 encoding
opt.encoding = "utf-8";
opt.fileencoding = "utf-8";
opt.autochdir = true;
opt.autowrite = true;
-- keymapping time
opt.ttimeout = true; -- separte mapping and keycode time
opt.timeoutlen = 300;
opt.ttimeoutlen = 20;
-- use undofile
opt.backup = false;
opt.swapfile = false;
opt.undofile = true;
opt.undodir = { fn.expand("~/.config") .. "/nvim/undo" };
-- filetype
vim.g.tex_flavor = "latex";
-- search
opt.incsearch = true;
opt.hlsearch = false;
-- sound
opt.visualbell = true;
-- style
cmd "syntax enable";
opt.background = "dark";
opt.termguicolors = true;
-- indentation
opt.expandtab = true;
opt.tabstop = 4;
opt.softtabstop = -1;
opt.shiftwidth = 4;
-- line number
opt.number = true;
opt.numberwidth = 4;
local numbertogglegroup = augroup("numbertoggle", { clear = true });
autocmd({
    "BufEnter",
    "FocusGained",
    "InsertLeave",
    "WinEnter",
}, {
    pattern = '*',
    callback = function()
        vim.opt.relativenumber = true;
    end,
    group = numbertogglegroup,
});
autocmd({
    "BufLeave",
    "FocusLost",
    "InsertEnter",
    "WinLeave",
}, {
    pattern = '*',
    callback = function()
        vim.opt.relativenumber = false;
    end,
    group = numbertogglegroup,
});

opt.clipboard = "unnamedplus";
vim.cmd([[
    let g:clipboard = {
    \	'name': 'win32yank-wsl',
    \	'copy': {
    \		'+': 'win32yank.exe -i --crlf',
    \		'*': 'win32yank.exe -i --crlf',
    \	},
    \	'paste': {
    \      '+': 'win32yank.exe -o --lf',
    \      '*': 'win32yank.exe -o --lf',
    \   },
    \	'cache_enabled': 1,
    \}
]]);
