{
  lib,
  pkgs,
  ...
}:
{
  programs.fastfetch = {
    enable = true;
    package = pkgs.fastfetch;

    settings = {
      "$schema" = "https://github.com/fastfetch-cli/fastfetch/raw/dev/doc/json_schema.json";
      logo = {
        source = "NixOS";
        width = 12;
        height = 12;
        padding = {
          top = 2;
          right = 2;
        };
      };
      modules = [
        {
          type = "custom";
          format = "{#89}┌──────────────────────Hardware──────────────────────┐";
        }
        {
          type = "host";
          key = " PC";
          keyColor = "green";
        }
        {
          type = "cpu";
          key = "│ ├ CPU";
          keyColor = "green";
        }
        {
          type = "gpu";
          key = "│ ├󰍛 GPU";
          keyColor = "green";
        }
        {
          type = "memory";
          key = "│ ├ Memory";
          keyColor = "green";
        }
        {
          type = "disk";
          key = "└ └ Disk";
          keyColor = "green";
        }
        {
          type = "custom";
          format = "{#89}└────────────────────────────────────────────────────┘";
        }
        "break"
        {
          type = "custom";
          format = "{#89}┌──────────────────────Software──────────────────────┐";
        }
        {
          type = "os";
          key = " OS";
          keyColor = "yellow";
        }
        {
          type = "kernel";
          key = "│ ├ Kernel";
          keyColor = "yellow";
        }
        {
          type = "packages";
          key = "└ └󰏖 Packages";
          keyColor = "yellow";
        }
        {
          type = "de";
          key = " DE";
          keyColor = "blue";
        }
        {
          type = "lm";
          key = "│ ├ LM";
          keyColor = "blue";
        }
        {
          type = "wm";
          key = "│ ├ WM";
          keyColor = "blue";
        }
        {
          type = "wmtheme";
          key = "│ ├󰉼";
          keyColor = "blue";
        }

        {
          type = "terminal";
          key = "│ ├ Terminal";
          keyColor = "blue";
        }
        {
          type = "shell";
          key = "└ └ Shell";
          keyColor = "blue";
        }
        {
          type = "custom";
          format = "{#89}└────────────────────────────────────────────────────┘";
        }
        "break"
        {
          type = "custom";
          format = "{#89}┌─────────────────Uptime / Age / DT-─────────────────┐";
        }
        {
          type = "command";
          key = "  OS Age ";
          keyColor = "magenta";
          text = "birth_install=$(stat -c %W /); current=$(date +%s); time_progression=$((current - birth_install)); days_difference=$((time_progression / 86400)); echo $days_difference days";
        }
        {
          type = "uptime";
          key = "  Uptime ";
          keyColor = "magenta";
        }
        {
          type = "datetime";
          key = "  DateTime ";
          keyColor = "magenta";
        }
        {
          type = "custom";
          format = "{#89}└────────────────────────────────────────────────────┘";
        }

        {
          type = "colors";
          paddingLeft = 2;
          symbol = "circle";
        }

      ];
    };

    # settings = {
    #   "$schema" = "https://github.com/fastfetch-cli/fastfetch/raw/dev/doc/json_schema.json";
    #   logo = {
    #     padding = {
    #       top = 2;
    #     };
    #   };
    #   display = {
    #     color = {
    #       keys = "green";
    #       title = "blue";
    #     };
    #     percent = {
    #       type = 9;
    #     };
    #     separator = " 󰁔 ";
    #   };
    #   modules =
    #     [
    #       {
    #         type = "custom";
    #         outputColor = "blue";
    #         format = ''┌──────────── OS Information ────────────┐'';
    #       }
    #       {
    #         type = "title";
    #         key = " ╭─ ";
    #         keyColor = "green";
    #         color = {
    #           user = "green";
    #           host = "green";
    #         };
    #       }
    #     ]
    #     ++ lib.optionals pkgs.stdenv.hostPlatform.isDarwin [
    #       {
    #         type = "os";
    #         key = " ├─  ";
    #         keyColor = "green";
    #       }
    #       {
    #         type = "kernel";
    #         key = " ├─  ";
    #         keyColor = "green";
    #       }
    #       {
    #         type = "packages";
    #         key = " ├─  ";
    #         keyColor = "green";
    #       }
    #     ]
    #     ++ lib.optionals pkgs.stdenv.hostPlatform.isLinux [
    #       {
    #         type = "os";
    #         key = " ├─ ";
    #         keyColor = "green";
    #       }
    #       {
    #         type = "kernel";
    #         key = " ├─ ";
    #         keyColor = "green";
    #       }
    #       {
    #         type = "packages";
    #         key = " ├─ ";
    #         keyColor = "green";
    #       }
    #     ]
    #     ++ [
    #       {
    #         type = "shell";
    #         key = " ╰─  ";
    #         keyColor = "green";
    #       }
    #       {
    #         type = "custom";
    #         outputColor = "blue";
    #         format = ''├───────── Hardware Information ─────────┤'';
    #       }
    #       {
    #         type = "display";
    #         key = " ╭─ 󰍹 ";
    #         keyColor = "blue";
    #         compactType = "original-with-refresh-rate";
    #       }
    #       {
    #         type = "cpu";
    #         key = " ├─ 󰍛 ";
    #         keyColor = "blue";
    #       }
    #       {
    #         type = "gpu";
    #         key = " ├─  ";
    #         keyColor = "blue";
    #       }
    #       {
    #         type = "disk";
    #         key = " ├─ 󱛟 ";
    #         keyColor = "blue";
    #       }
    #       {
    #         type = "memory";
    #         key = " ╰─  ";
    #         keyColor = "blue";
    #       }
    #       {
    #         type = "custom";
    #         outputColor = "blue";
    #         format = ''├───────── Software Information ─────────┤'';
    #       }
    #       {
    #         type = "wm";
    #         key = " ╭─  ";
    #         keyColor = "yellow";
    #       }
    #       {
    #         type = "terminal";
    #         key = " ├─  ";
    #         keyColor = "yellow";
    #       }
    #       {
    #         type = "font";
    #         key = " ╰─  ";
    #         keyColor = "yellow";
    #       }
    #       {
    #         type = "custom";
    #         outputColor = "blue";
    #         format = ''└────────────────────────────────────────┘'';
    #       }
    #       {
    #         type = "custom";
    #         format = "   {#39}   {#34}    {#36}    {#35}    {#34}    {#33}    {#32}    {#31} ";
    #       }
    #       "break"
    #     ];
    # };
  };
}
