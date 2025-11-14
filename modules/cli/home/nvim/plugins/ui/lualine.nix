{
  inputs,
  pkgs,
  ...
}:
{
  programs.nixvim = {
    extraPlugins = with pkgs.vimUtils; [
      (buildVimPlugin {
        pname = "copilot-lualine";
        version = "2025-11-14";
        src = inputs.copilot-lualine-nvim;
      })
    ];
    plugins = {
      web-devicons.enable = true;
      # https://nix-community.github.io/nixvim/plugins/lualine/index.html
      lualine = {
        enable = true;
        settings.options = {
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
  };
}
