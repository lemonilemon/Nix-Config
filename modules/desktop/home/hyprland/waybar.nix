{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    home.packages = with pkgs; [
      htop
      libnotify
      libappindicator # Library for appindicator support
    ];
    programs.foot = {
      enable = true;
      settings = {
        main = {
          term = "xterm-256color";
        };
      };
    };

    # Status bar
    programs.waybar = {
      enable = true;
      systemd = {
        enable = true;
        targets = [ "hyprland-session.target" ];
      };
      settings = {
        mainBar = {
          layer = "top";
          position = "top";
          spacing = 4;
          margin-top = 8;
          margin-left = 8;
          margin-right = 8;
          modules-left = [
            "hyprland/workspaces"
            "hyprland/submap"
            "mpris"
          ];
          modules-center = [
            "clock"
          ];
          modules-right = [
            "tray"
            "idle_inhibitor"
            "temperature"
            "cpu"
            "memory"
            "bluetooth"
            "network"
            "pulseaudio"
            "battery"
            "custom/power"
          ];

          "hyprland/workspaces" = {
            format = "{icon}";
            "format-icons" = {
              "active" = "";
              "default" = "";
              "empty" = "";
            };
            "persistent-workspaces" = {
              "*" = [
                1
                2
                3
                4
                5
              ];
            };
          };

          "hyprland/submap" = {
            format = "<span style=\"italic\">{}</span>";
          };

          mpris = {
            interval = 1;
            format = "{player_icon} {dynamic}";
            format-paused = "{status_icon} <i>{dynamic}</i>";
            "max-length" = 40;
            dynamic-order = [
              "artist"
              "title"
            ];
            player-icons = {
              default = "";
              tidal-hifi = "";
              cmus = "";
            };
            status-icons = {
              paused = "";
            };
          };

          clock = {
            interval = 60;
            timezone = "Asia/Taipei";
            format = " {:%H:%M}";
            "format-alt" = " {:%F}";
            "tooltip-format" = "<tt><small>{calendar}</small></tt>";
            calendar = {
              mode = "year";
              "mode-mon-col" = 3;
              "weeks-pos" = "right";
              "on-scroll" = 1;
              "on-click-right" = "mode";
              format = {
                months = "<span color='#cba6f7'><b>{}</b></span>";
                days = "<span color='#cdd6f4'><b>{}</b></span>";
                weeks = "<span color='#94e2d5'>W{}</span>";
                weekdays = "<span color='#f9e2af'><b>{}</b></span>";
                today = "<span color='#f5e0dc'><b><u>{}</u></b></span>";
              };
            };
            actions = {
              "on-click-right" = "mode";
              "on-click-forward" = "tz_up";
              "on-click-backward" = "tz_down";
              "on-scroll-up" = "shift_up";
              "on-scroll-down" = "shift_down";
            };
          };

          tray = {
            "icon-size" = 14;
            "spacing" = 8;
          };

          idle_inhibitor = {
            format = "{icon}";
            "format-icons" = {
              activated = "";
              deactivated = "";
            };
          };

          temperature = {
            interval = 3;
            "hwmon-path" = "/sys/class/hwmon/hwmon1/temp1_input";
            "critical-threshold" = 90;
            "format-critical" = " {temperatureC}°C";
            format = "{icon} {temperatureC}°C";
            "format-icons" = [
              ""
              ""
              ""
            ];
          };

          cpu = {
            interval = 3;
            format = " {usage}%";
            "on-click" = "kitty -- htop";
          };

          memory = {
            interval = 3;
            format = " {}%";
            "on-click" = "kitty -- htop";
            "tooltip-format" = "Used: {used:0.1f}G/{total:0.1f}G\nSwap: {swapUsed:0.1f}G/{swapTotal:0.1f}G";
            states = {
              critical = 80;
            };
          };

          bluetooth = {
            "format-on" = "";
            "format-off" = "";
            "format-connected" = "󰂯 {device_alias}";
            "format-connected-battery" = "󰂯 {device_alias} {device_battery_percentage}%";
            "format-device-preference" = [
              "ugreen_1"
              "ugreen_2"
            ];
            "tooltip-format" = "{controller_alias}\t{controller_address}\n\n{num_connections} connected";
            "tooltip-format-connected" =
              "{controller_alias}\t{controller_address}\n\n{num_connections} connected\n\n{device_enumerate}";
            "tooltip-format-enumerate-connected" = "{device_alias}\t{device_address}";
            "tooltip-format-enumerate-connected-battery" =
              "{device_alias}\t{device_address}\t{device_battery_percentage}%";
          };

          network = {
            interval = 60;
            "interface-ethernet" = "enp1s*";
            "interface-wifi" = "wlan0";
            "format-ethernet" = "󰌗 {ifname}";
            "format-wifi" = " {essid} {signalStrength}%";
            "tooltip-format-ethernet" = "{ifname}: {ipaddr}/{cidr}";
            "tooltip-format-wifi" = "{ifname}: {ipaddr}/{cidr}";
            "format-linked" = "󰈀 (No IP)";
            "format-disconnected" = "⚠ Disconnected";
          };

          pulseaudio = {
            "scroll-step" = 2;
            format = "{icon} {volume}%";
            "format-bluetooth" = "󰂯 {icon} {volume}%";
            "format-bluetooth-muted" = "󰂲 ";
            "format-muted" = "󰝟 ";
            "format-source" = " {volume}%";
            "format-source-muted" = " ";
            "format-icons" = {
              headphone = " ";
              "hands-free" = " ";
              headset = " ";
              phone = "";
              portable = "";
              car = "";
              default = [
                ""
                ""
                ""
              ];
            };
            "on-click" = "foot --app-id pulsemixer pulsemixer";
          };

          battery = {
            interval = 30;
            states = {
              good = 95;
              warning = 30;
              critical = 20;
            };
            format = "{icon} {capacity}%";
            "format-charging" = "󰂄 {capacity}%";
            "format-plugged" = "󰂄 {capacity}%";
            "format-alt" = "{icon} {time}";
            "format-icons" = [
              "󰁻"
              "󰁼"
              "󰁾"
              "󰂀"
              "󰂂"
              "󰁹"
            ];
          };

          "custom/power" = {
            format = "";
            "on-click" = "wlogout";
            justify = "center";
            tooltip = false;
          };

          "custom/kernel" = {
            exec = "uname -r | sed -E 's/^([0-9]+\\.[0-9]+\\.[0-9]+)-.*-([a-zA-Z0-9]+)/\\1-\\2/'";
            format = " {}";
          };

          "disk#ssd" = {
            interval = 60;
            format = " {free}";
            path = "/";
            tooltip = true;
            warning = 80;
            critical = 90;
          };
        };
      };

      style = ''
        @define-color rosewater #f5e0dc;
        @define-color flamingo #f2cdcd;
        @define-color pink #f5c2e7;
        @define-color mauve #cba6f7;
        @define-color red #f38ba8;
        @define-color maroon #eba0ac;
        @define-color peach #fab387;
        @define-color yellow #f9e2af;
        @define-color green #a6e3a1;
        @define-color teal #94e2d5;
        @define-color sky #89dceb;
        @define-color sapphire #74c7ec;
        @define-color blue #89b4fa;
        @define-color lavender #b4befe;
        @define-color text #cdd6f4;
        @define-color subtext1 #bac2de;
        @define-color subtext0 #a6adc8;
        @define-color overlay2 #9399b2;
        @define-color overlay1 #7f849c;
        @define-color overlay0 #6c7086;
        @define-color surface2 #585b70;
        @define-color surface1 #45475a;
        @define-color surface0 #313244;
        @define-color base #1e1e2e;
        @define-color mantle #181825;
        @define-color crust #11111b;

        * {
          border: none;
          border-radius: 0;
          font-family: "JetBrainsMono Nerd Font Propo", "Font Awesome 6 Free";
          font-size: 13px;
          min-height: 0;
          transition: color 0.2s ease, background-color 0.2s ease;
        }

        window#waybar {
          background-color: @mantle;
          border-radius: 12px;
          color: @text;
        }

        window#waybar.hidden {
          opacity: 0.2;
        }

        tooltip {
          background-color: @base;
          border: 1px solid @surface1;
          border-radius: 8px;
        }

        tooltip label {
          color: @text;
        }

        button {
          box-shadow: none;
          border: none;
          border-radius: 8px;
        }

        button:hover {
          background: @surface0;
        }

        /* Workspaces */

        #workspaces {
          padding: 0 4px;
        }

        #workspaces button {
          padding: 4px 8px;
          background-color: transparent;
          color: @subtext0;
          border-radius: 8px;
        }

        #workspaces button:hover {
          background-color: @surface0;
          color: @text;
        }

        #workspaces button.active {
          background-color: @surface0;
          color: @green;
        }

        #workspaces button.urgent {
          color: @red;
        }

        /* All modules */

        #mpris,
        #clock,
        #pulseaudio,
        #bluetooth,
        #network,
        #memory,
        #cpu,
        #temperature,
        #disk,
        #custom-power,
        #idle_inhibitor,
        #battery,
        #submap,
        #tray {
          padding: 4px 10px;
          color: @text;
        }

        #window,
        #workspaces {
          margin: 0 4px;
        }

        /* Module accent colors */

        #clock {
          color: @green;
          font-weight: bold;
          padding: 4px 14px;
        }

        #battery {
          color: @blue;
        }

        #battery.warning {
          color: @yellow;
        }

        #battery.critical {
          color: @red;
          animation-name: blink;
          animation-duration: 0.5s;
          animation-timing-function: linear;
          animation-iteration-count: infinite;
          animation-direction: alternate;
        }

        #battery.charging,
        #battery.plugged {
          color: @green;
        }

        @keyframes blink {
          to {
            color: @overlay1;
          }
        }

        #cpu {
          color: @mauve;
        }

        #memory {
          color: @maroon;
        }

        #memory.critical {
          color: @red;
        }

        #temperature {
          color: @pink;
        }

        #temperature.critical {
          color: @red;
        }

        #network {
          color: @peach;
        }

        #network.disconnected {
          color: @red;
        }

        #bluetooth {
          color: @sapphire;
        }

        #bluetooth.off {
          color: @overlay1;
        }

        #pulseaudio {
          color: @yellow;
        }

        #pulseaudio.muted {
          color: @overlay1;
        }

        #mpris {
          color: @teal;
          padding: 4px 12px;
        }

        #submap {
          color: @yellow;
          font-style: italic;
        }

        #tray {
          padding: 4px 8px;
        }

        #tray > .passive {
          -gtk-icon-effect: dim;
        }

        #tray > .needs-attention {
          -gtk-icon-effect: highlight;
        }

        #idle_inhibitor {
          color: @overlay1;
        }

        #idle_inhibitor.activated {
          color: @yellow;
        }

        #custom-power {
          color: @red;
          padding: 4px 14px;
          font-size: 14px;
          border-radius: 0 12px 12px 0;
        }

        #custom-power:hover {
          background-color: @red;
          color: @mantle;
        }
      '';
    };
  };
}
