{
  config,
  lib,
  ...
}:
let
  helpers = config.lib.nixvim;
in
{
  programs.nixvim = {
    plugins.edgy = {
      enable = lib.mkDefault true;
      settings = {
        bottom = [
          {
            ft = "toggleterm";
            filter = helpers.mkRaw ''
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
