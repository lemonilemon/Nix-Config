{
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
        nil-ls = {
          enable = true;
          settings = {
            nix.flake.autoArchive = true;
            formatting.command = [
              "nixpkgs-fmt"
            ];
          };
        };
        nixd = {
          enable = true;
          settings = {
            formatting.command = [ "nixpkgs-fmt" ];
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
      };
    };
  };
}
