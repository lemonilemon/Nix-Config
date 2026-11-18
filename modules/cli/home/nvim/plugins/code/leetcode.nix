{ config, lib, ... }:
{
  config = lib.mkIf (config.home.cli.nvim.enable && config.home.cli.nvim.leetcode.enable) {
    programs.nixvim.plugins.leetcode = {
      enable = true;
      settings = {
        lang = "python3";
        hooks.enter = [
          {
            __raw = ''
              function()
                local names = vim.tbl_map(function(c)
                  return c.name
                end, vim.lsp.get_configs({ enabled = true }))
                vim.lsp.enable(names, false)
                require("cmp").setup({ enabled = false })
                require("copilot.command").disable()
                require("conform").setup({ format_on_save = false })
              end
            '';
          }
        ];
      };
    };

    home.shellAliases.leet = "nvim leetcode.nvim";
  };
}
