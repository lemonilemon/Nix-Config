{ pkgs
, ...
}: {
  programs.nixvim = {
    extraPlugins = with pkgs.vimUtils; [
      (buildVimPlugin {
        pname = "copilot-lualine";
        version = "2024-09-08";
        src = pkgs.fetchFromGitHub {
          owner = "AndreM222";
          repo = "copilot-lualine";
          rev = "main";
          sha256 = "sha256-PXiJ7rdlE8J93TFtu+D+8398Wg7DhK7EZ0Aw4JDoqWM=";
        };

      })
    ];
    plugins = {
      # https://nix-community.github.io/nixvim/plugins/lualine/index.html
      lualine = {
        enable = true;
        iconsEnabled = true;
        theme = "auto";
        globalstatus = true;
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
            "diagnostics"
          ];
          lualine_c = [ "filename" ];
          lualine_x = [
            {
              name = "copilot";
              extraConfig = {
                show_colors = true;
              };
            }
            "encoding"
            "fileformat"
            "filetype"
          ];
          lualine_y = [ "progress" ];
          lualine_z = [ "location" ];
        };
        inactiveSections = {
          lualine_a = [ ];
          lualine_b = [ ];
          lualine_c = [ "filename" ];
          lualine_x = [ "location" ];
          lualine_y = [ ];
          lualine_z = [ ];
        };
      };
    };
  };
}
