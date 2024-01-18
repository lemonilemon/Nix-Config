local api = vim.api;
local map = api.nvim_set_keymap;
local mapopt = { noremap = true, silent = true }
map("n", " ", "<Nop>", mapopt)
vim.g.mapleader = " "
vim.g.maplocalleader = " "

map('i', "jj", "<ESC>", mapopt)
map('n', "<F6>", ":RunCode<CR>", mapopt)
map('n', "<F7>", ":Template ", { noremap = true })
map('n', "<F8>", ":LazyGit<CR>", mapopt)

map('n', 'tt', ':NvimTreeFocus<CR>', mapopt)
map('n', '<leader>ff', ":Telescope find_files<CR>", mapopt)
map('n', '<leader>fg', ":Telescope live_grep<CR>", mapopt)
map('n', '<leader>fb', ":Telescope buffers<CR>", mapopt)
map('n', '<leader>fh', ":Telescope help_tags<CR>", mapopt)
map('n', '<leader>ft', ":Telescope find_template<CR>", mapopt)

