{ pkgs, ... }:
{
  home.packages = with pkgs; [
    nixfmt-rfc-style
    prettierd
    stylua
    isort
    black
  ];
  programs.nixvim.plugins = {
    conform-nvim = {
      enable = true;
      settings = {
        formatters = {
          black = {
            prepend_args = [ "--fast" ];
          };
        };
        format_on_save = {
          lspFallback = true;
          timeoutMs = 500;
        };
        notify_on_error = true;
        formatters_by_ft = {
          html = {
            __unkeyed-1 = "prettierd";
            __unkeyed-2 = "prettier";
            stop_after_first = true;
          };
          css = {
            __unkeyed-1 = "prettierd";
            __unkeyed-2 = "prettier";
            stop_after_first = true;
          };
          javascript = {
            __unkeyed-1 = "prettierd";
            __unkeyed-2 = "prettier";
            stop_after_first = true;
          };
          javascriptreact = {
            __unkeyed-1 = "prettierd";
            __unkeyed-2 = "prettier";
            stop_after_first = true;
          };
          typescript = {
            __unkeyed-1 = "prettierd";
            __unkeyed-2 = "prettier";
            stop_after_first = true;
          };

          typescriptreact = {
            __unkeyed-1 = "prettierd";
            __unkeyed-2 = "prettier";
            stop_after_first = true;
          };
          python = [
            "isort"
            "black"
          ];
          lua = [ "stylua" ];
          nix = [ "nixfmt" ];
          # markdown = [
          #   [
          #     "prettierd"
          #     "prettier"
          #   ]
          # ];
          yaml = [
            "yamllint"
            "yamlfmt"
          ];
          tex = [ "latexindent" ];
          "_" = [
            "squeeze_blanks"
            "trim_whitespace"
            "trim_newlines"
          ];
        };
      };
    };
  };
}
