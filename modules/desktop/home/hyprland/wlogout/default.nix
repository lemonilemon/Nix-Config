{
  lib,
  config,
  ...
}:
{
  imports = [
    ../hyprlock.nix
  ];
  # Icons
  config = lib.mkIf config.home.desktop.hyprland.enable {
    home.file.".config/wlogout/icons" = {
      enable = true;
      source = ./icons;
      recursive = true;
    };
    # Logout menu
    programs.wlogout = {
      enable = true;
      style = ''
        * {
          box-shadow: none;
        }

        window {
          background-color: rgba(30, 30, 46, 0.9);
          background-image: none;
        }

        button {
          color: #cdd6f4;
          background-color: rgba(49, 50, 68, 0.8);
          border: 2px solid rgba(69, 71, 90, 0.5);
          border-radius: 16px;
          background-repeat: no-repeat;
          background-position: center 35%;
          background-size: 38%;
          font-size: 13px;
          font-family: "JetBrains Mono Nerd Font";
          margin: 8px;
          padding-top: 20px;
          transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease;
        }

        button:focus {
          outline: none;
        }

        #lock:hover {
          background-color: rgba(137, 180, 250, 0.2);
          border-color: #89b4fa;
          color: #89b4fa;
        }

        #logout:hover {
          background-color: rgba(249, 226, 175, 0.2);
          border-color: #f9e2af;
          color: #f9e2af;
        }

        #suspend:hover {
          background-color: rgba(148, 226, 213, 0.2);
          border-color: #94e2d5;
          color: #94e2d5;
        }

        #hibernate:hover {
          background-color: rgba(203, 166, 247, 0.2);
          border-color: #cba6f7;
          color: #cba6f7;
        }

        #reboot:hover {
          background-color: rgba(250, 179, 135, 0.2);
          border-color: #fab387;
          color: #fab387;
        }

        #shutdown:hover {
          background-color: rgba(243, 139, 168, 0.2);
          border-color: #f38ba8;
          color: #f38ba8;
        }
      '';
      layout = [
        {
          "label" = "lock";
          "action" = "hyprlock -q";
          "text" = "Lock";
          "keybind" = "l";
        }
        {
          "label" = "hibernate";
          "action" = "systemctl hibernate";
          "text" = "Hibernate";
          "keybind" = "h";
        }
        {
          "label" = "logout";
          "action" = "hyprctl dispatch exit";
          "text" = "Logout";
          "keybind" = "e";
        }
        {
          "label" = "shutdown";
          "action" = "systemctl poweroff";
          "text" = "Shutdown";
          "keybind" = "s";
        }
        {
          "label" = "suspend";
          "action" = "systemctl suspend";
          "text" = "Suspend";
          "keybind" = "u";
        }
        {
          "label" = "reboot";
          "action" = "systemctl reboot";
          "text" = "Reboot";
          "keybind" = "r";
        }
      ];
    };
  };
}
