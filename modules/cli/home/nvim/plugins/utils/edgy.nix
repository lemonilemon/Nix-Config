{
  config,
  lib,
  ...
}:
{
  programs.nixvim = {
    plugins.edgy = {
      enable = lib.mkDefault true;
      settings = {
        bottom = [
          {
            ft = "toggleterm";
            filter = config.lib.nixvim.mkRaw ''
              function(buf, win)
                return vim.api.nvim_win_get_config(win).relative == ""
              end
            '';
          }
        ];
      };
    };
  };
}
