{
    programs.nixvim = {
        plugins.transparent = {
            # https://nix-community.github.io/nixvim/plugins/transparent/index.html
            enable = true;
            settings = {
                extra_groups = [
                    "Pmenu" # Popup menu (e.g. cmp)
                    "Float" 
                    "NormalFloat"
                    "Notify" # Notify
                    "WhichKey" # WhichKey
                    "NvimTreeNormal"# NvimTree
                    "NvimTreeNormalNC"
                    "NvimTreeWinSeparator"
                    "NvimTreeCursorLine"
                    "NvimTreeCursorColumn"
                    "TelescopeNormal" # Telescope
                    "TelescopeBorder"
                    "LspFloatWinNormal" # Lsp
                    "LspFloatWinBorder"
                    "DiagnosticVirtualTextError" # Diagnostic
                    "DiagnosticVirtualTextWarn"
                    "DiagnosticVirtualTextInfo"
                    "DignosticVirtualTextHint"
                ];
            };
        };
        globals = {
            transparent_enabled = true;
        };
    };
}
