vim.g.mapleader = " "
vim.g.maplocalleader = " "
local map = vim.api.nvim_set_keymap
local opt = { noremap = true, silent = true }
map('i', 'jj', '<ESC>', opt)
map('n', '<F5>', ':NvimTreeToggle<CR>', opt)
map('n', 'tt', ':NvimTreeFocus<CR>', opt)
map('n', '<F6>', ':RunCode<CR>', opt)
vim.keymap.set('n', '<F7>', function()
    vim.fn.feedkeys(':Template ')
end, { remap = true})
