{ pkgs, ... }:
{
  # https://nix-community.github.io/nixvim/plugins/conform-nvim/index.html
  home.packages = with pkgs; [ nixfmt-rfc-style ];
  programs.nixvim.plugins = {
    conform-nvim = {
      enable = true;
      settings = {
        format_on_save = {
          lspFallback = true;
          timeoutMs = 500;
        };
        notify_on_error = true;
        formatters_by_ft = {
          liquidsoap = [ "liquidsoap-prettier" ];
          html = [
            [
              "prettierd"
              "prettier"
            ]
          ];
          css = [
            [
              "prettierd"
              "prettier"
            ]
          ];
          javascript = [
            [
              "prettierd"
              "prettier"
            ]
          ];
          javascriptreact = [
            [
              "prettierd"
              "prettier"
            ]
          ];
          typescript = [
            [
              "prettierd"
              "prettier"
            ]
          ];
          typescriptreact = [
            [
              "prettierd"
              "prettier"
            ]
          ];
          python = [ "black" ];
          lua = [ "stylua" ];
          nix = [ "nixfmt" ];
          markdown = [
            [
              "prettierd"
              "prettier"
            ]
          ];
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
