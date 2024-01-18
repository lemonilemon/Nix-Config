return {
    "nvim-treesitter/nvim-treesitter",
    main = "nvim-treesitter.configs",
    opts = {
        ensure_installed = {
          "c",
          "cpp",
          "lua",
          "vim",
          "vimdoc",
          "query",
          "python",
        },
        sync_install = true,
        auto_install = true,
        ignore_install = {},
        highlight = {
            enable = true,
            disable = {
                "latex", -- vimtex
            },
            additional_vim_regex_highlighting = { "markdown", "latex" },
        },
        indent = {
            enable = true,
        },
    },
    build = ":TSUpdate",
}

