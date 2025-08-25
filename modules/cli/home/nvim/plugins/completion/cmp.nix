{
  programs.nixvim = {
    plugins = {
      cmp-emoji = {
        enable = true;
      };
      copilot-cmp = {
        enable = true;
      };
      copilot-lua = {
        enable = true;
        settings = {
          suggestion = {
            enabled = false;
          };
          panel = {
            enabled = false;
          };
          filetypes = {
            markdown = true;
            help = true;
          };
        };
      };
      cmp = {
        enable = true;
        settings = {
          autoEnableSources = true;
          experimental = {
            ghost_text = true;
          };
          performance = {
            debounce = 60;
            fetchingTimeout = 200;
            maxViewEntries = 30;
          };
          snippet = {
            expand = "luasnip";
          };
          formatting = {
            fields = [
              "kind"
              "abbr"
              "menu"
            ];
          };
          sources = [
            { name = "nvim_lsp"; }
            { name = "path"; }
            { name = "buffer"; }
            { name = "copilot"; }
            { name = "emoji"; }
            { name = "luasnip"; }
            { name = "rg"; }
            { name = "nvim_lua"; }
          ];
          mapping = {
            "<Tab>" = "cmp.mapping(cmp.mapping.select_next_item(), {'i', 's'})";
            "<C-j>" = "cmp.mapping.select_next_item()";
            "<C-k>" = "cmp.mapping.select_prev_item()";
            "<C-b>" = "cmp.mapping.scroll_docs(-4)";
            "<C-f>" = "cmp.mapping.scroll_docs(4)";
            "<C-CR>" = "cmp.mapping.abort()";
            "<C-Space>" = "cmp.mapping.complete()";
            "<CR>" = "cmp.mapping.confirm({ select = true })";
          };
          window = {
            completion = {
              border = "solid";
            };
            documentation = {
              border = "solid";
            };
          };
        };
      };
    };
    extraConfigLua = ''
      luasnip = require("luasnip")
      kind_icons = {
          Text = "󰊄",
          Method = "",
          Function = "󰡱",
          Constructor = "",
          Field = "",
          Variable = "󱀍",
          Class = "",
          Interface = "",
          Module = "󰕳",
          Property = "",
          Unit = "",
          Value = "",
          Enum = "",
          Keyword = "",
          Snippet = "",
          Color = "",
          File = "",
          Reference = "",
          Folder = "",
          EnumMember = "",
          Constant = "",
          Struct = "",
          Event = "",
          Operator = "",
          TypeParameter = "",
      } 

      local cmp = require'cmp'

      -- Use buffer source for `/` (if you enabled `native_menu`, this won't work anymore).
      cmp.setup.cmdline({'/', "?" }, {
          sources = {
              { name = 'buffer' }
          }
      })

      -- Set configuration for specific filetype.
      cmp.setup.filetype('gitcommit', {
          sources = cmp.config.sources({
              { name = 'cmp_git' }, -- You can specify the `cmp_git` source if you were installed it.
          }, {
                  { name = 'buffer' },
              })
      })

      -- Use cmdline & path source for ':' (if you enabled `native_menu`, this won't work anymore).
      cmp.setup.cmdline(':', {
          sources = cmp.config.sources({
              { name = 'path' }
          }, {
                  { name = 'cmdline' }
              }),
          --      formatting = {
          --       format = function(_, vim_item)
          --         vim_item.kind = cmdIcons[vim_item.kind] or "FOO"
          --       return vim_item
          --      end
          -- }
      })  '';
  };
}
