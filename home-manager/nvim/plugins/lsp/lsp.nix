{ pkgs, ... }:
{

  home.packages = with pkgs; [ nixfmt-rfc-style ];
  programs.nixvim.plugins = {
    # https://nix-community.github.io/nixvim/plugins/lsp-format/index.html
    lsp-format = {
      enable = true;
    };
    # https://nix-community.github.io/nixvim/plugins/lsp/index.html
    lsp = {
      enable = true;
      servers = {
        eslint = {
          enable = true;
        };
        html = {
          enable = true;
        };
        lua-ls = {
          enable = true;
          settings = {
            completion = {
              callSnippet = "Replace";
            };
            diagnostics = {
              globals = [ "vim" ];
            };
          };
        };
        ccls = {
          enable = true;
          initOptions = {
            clang = {
              extraArgs = [ "-Wno-c++17-extensions" ];
            };
          };
        };
        nixd = {
          enable = true;
          settings = {
            formatting.command = [ "nixfmt-rfc-style" ];
          };
        };
        marksman = {
          enable = true;
        };
        pyright = {
          enable = true;
        };
        gopls = {
          enable = true;
        };
        terraformls = {
          enable = true;
        };
        ts-ls = {
          enable = false;
        };
        yamlls = {
          enable = true;
        };
        texlab = {
          enable = true;
        };
      };
    };
  };
}
