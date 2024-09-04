{
    # https://nix-community.github.io/nixvim/plugins/noice/index.html
    plugins.noice = {
        enable = true;
        presets = {
            bottom_search = true;
            command_palette = true;
            long_message_to_split = true;
            inc_rename = false;
            lsp_doc_border = false;
        };
        notify = {
            enabled = false;
        };
        messages = {
            enabled = true; 
        };
        lsp = {
            message = {
                enabled = true;
            };
            progress = {
                enabled = false;
                view = "mini";
            };
        };
        popupmenu = {
            enabled = true;
            backend = "nui";
        };
        format = {
            filter = {
                pattern = [
                    ":%s*%%s*s:%s*"
                    ":%s*%%s*s!%s*"
                    ":%s*%%s*s/%s*"
                    "%s*s:%s*"
                    ":%s*s!%s*"
                    ":%s*s/%s*"
                ];
                icon = "";
                lang = "regex";
            };
            replace = {
                pattern = [
                    ":%s*%%s*s:%w*:%s*"
                    ":%s*%%s*s!%w*!%s*"
                    ":%s*%%s*s/%w*/%s*"
                    "%s*s:%w*:%s*"
                    ":%s*s!%w*!%s*"
                    ":%s*s/%w*/%s*"
                ];
                icon = "󱞪";
                lang = "regex";
            };
        };
    };
}
