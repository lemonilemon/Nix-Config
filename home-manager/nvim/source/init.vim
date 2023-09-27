"" Plugins
set nocompatible " be iMproved, required 

call plug#begin('~/.config/nvim/plugged')
" The default plugin directory will be as follows:
"   - Vim (Linux/macOS): '~/.vim/plugged'
"   - Vim (Windows): '~/vimfiles/plugged'
"   - Neovim (Linux/macOS/Windows): stdpath('data') . '/plugged'
" You can specify a custom plugin directory by passing it as the argument
"   - e.g. `call plug#begin('~/.vim/plugged')`
"   - Avoid using standard Vim directory names like 'plugin'

" icon
Plug 'nvim-tree/nvim-web-devicons'
" theme
Plug 'xiyaowong/nvim-transparent'
Plug 'catppuccin/nvim', { 'as': 'catppuccin' }
" airline
Plug 'nvim-lualine/lualine.nvim'
" telescope
Plug 'nvim-lua/plenary.nvim'
Plug 'nvim-telescope/telescope.nvim', { 'branch': '0.1.x' }
Plug 'nvim-telescope/telescope-fzf-native.nvim', { 'do': 'make' }
" filetree
Plug 'nvim-tree/nvim-tree.lua'
" lsp service
Plug 'neovim/nvim-lspconfig'
" code runner
Plug 'CRAG666/code_runner.nvim'
" treesitter
Plug 'nvim-treesitter/nvim-treesitter'
" template
Plug 'nvimdev/template.nvim'
call plug#end()

"" Require plugin configs
lua require("lemonilemon")

"" auto install
autocmd VimEnter *
  \  if !empty(filter(copy(g:plugs), '!isdirectory(v:val.dir)'))
  \|   PlugInstall | q
  \| endif

"" Basic 
	" auto indentation
	set ai 
	" ignore case
	set ic 

	set enc=utf8 
	set mouse=a 
	set autochdir 
	set autowrite 
	set nobackup
"" delays
	set timeout ttimeout " separte mapping and keycode time
	set timeoutlen=500 " 500ms
	set ttimeoutlen=20 " 20ms
"" files
	set noswapfile
	set undofile
	set undodir=~/.config/nvim/undodir
"" search
	set incsearch
	set nohlsearch
"" Sound
	set visualbell
"" Style 
	syntax enable
	set background=dark
	let g:transparent_enabled = v:true
	hi Normal guibg=None ctermbg=None
    hi NonText guibg=None ctermbg=None
	set termguicolors
	hi CursorLine cterm=underline
"" Indentation 
	set expandtab
	set tabstop=4
	set softtabstop=-1
	set shiftwidth=4
	set smartindent

"" Line number 
	" add line number
	set number 
	" columns used for the line number
	set numberwidth=4
	augroup numbertoggle
		autocmd! 
		autocmd BufEnter,FocusGained,InsertLeave,WinEnter * if &nu && mode() != "i" | set rnu | endif 
		autocmd BufLeave,FocusLost,InsertEnter,WinLeave * if &nu | set nornu | endif
	augroup END
"" Clipboard
	set clipboard+=unnamedplus
	let g:clipboard = {
	\	'name': 'wsl-clip',
	\	'copy': {
	\		'+': 'clip.exe',
	\		'*': 'clip.exe',
	\	},
	\	'paste': {
	\      '+': 'powershell.exe -c [Console]::Out.Write($(Get-Clipboard -Raw).tostring().replace("`r", ""))',
	\      '*': 'powershell.exe -c [Console]::Out.Write($(Get-Clipboard -Raw).tostring().replace("`r", ""))',
	\   },
	\	'cache_enabled': 0,
	\}

