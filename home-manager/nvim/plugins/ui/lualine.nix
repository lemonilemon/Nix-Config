{
    # https://nix-community.github.io/nixvim/plugins/lualine/index.html
    /*  TODO: 
        - Add copilot icons to lualine
    extraPlugins = with pkgs.vimPlugins; [
        (buildVimPlugin {
            pname = "copilot-lualine";
        })
    ]; */
    plugins.lualine = {
        enable = true;
        icons_enabled = true;
        theme = "auto";
        componentSeparators = { 
            left = "";
            right = "";
        };
        sectionSeparators = {
            left = "";
            right = "";
        };
        alwaysDivideMiddle = true;
        sections = {
            lualine_a = [ "mode" ];
            lualine_b = [ 
                "branch" 
                "diff"
                {
                    name = "diagnostics";
                    extraConfig = {
                        sources = "nvim_diagnostic";
                        symbols = { 
                            error = " "; 
                            warn = " "; 
                            info = " "; 
                            hint = ""; 
                        };
                    };
                }
            ];
            lualine_c = [ "filename" ];
            lualine_x = [ 
                # TODO: Add copilot icon
                /* 
                {
                    name = "copilot" ;
                    extraConfig = {
                        show_colors = true;
                    }; 
                }
                */
                "encoding" 
                "fileformat" 
                "filetype" 
            ];
            lualine_y = [ "progress" ];
            lualine_z = [ "location" ];
        };
        inactiveSections = {
            lualine_a = [];
            lualine_b = [];
            lualine_c = [ "filename" ];
            lualine_x = [ "location" ];
            lualine_y = [];
            lualine_z = [];
        };
    };
}
