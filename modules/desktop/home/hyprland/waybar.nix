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
      # systemd.enable = true;
      settings = {
        mainBar = {
          layer = "top";
          position = "top";
          # height = 30;
          # width = 2560;
          spacing = 1;
          modules-left = [
            "hyprland/workspaces"
            "tray"
            "hyprland/submap"
            "mpris"
          ];
          modules-center = [
            # "hyprland/window"
          ];
          modules-right = [
            "idle_inhibitor"
            # "custom/kernel"
            # "disk#ssd"
            "temperature"
            "cpu"
            "memory"
            "bluetooth"
            "network"
            "pulseaudio"
            "clock"
            "battery"
            "custom/power"
          ];

          "hyprland/workspaces" = {
            format = "{icon}";
            "format-icons" = {
              "active" = "";
              "default" = "";
              "empty" = "";
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

          bluetooth = {
            "format-on" = "BT ";
            "format-off" = "({status}) ";
            "format-connected" = "{device_alias} ";
            "format-connected-battery" = "{device_alias} [{device_battery_percentage}%] ";
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
            "on-click" = "blueman-manager";
          };

          "hyprland/submap" = {
            format = "<span style=\"italic\">{}</span>";
          };

          mpris = {
            interval = 0;
            format = "{player_icon} {dynamic}";
            format-paused = "{status_icon} {dynamic}";
            dynamic-order = [
              "artist"
              "title"
            ];
            player-icons = {
              default = "";
              tidal-hifi = "🎵";
              cmus = "🎵";
            };
            status-icons = {
              paused = "▶";
            };
          };

          # "hyprland/scratchpad" = {
          #   format = "{icon} {count}";
          #   "show-empty" = false;
          #   "format-icons" = [
          #     ""
          #     ""
          #   ];
          #   tooltip = true;
          #   "tooltip-format" = "{app}: {title}";
          # };

          idle_inhibitor = {
            format = "{icon}";
            "format-icons" = {
              activated = "";
              deactivated = "";
            };
          };

          tray = {
            "icon-size" = 13;
            "spacing" = 10;
            # "show-passive-items" = true;
          };

          clock = {
            interval = 60;
            timezone = "Asia/Taipei";
            format = "{:%F %R }";
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

          cpu = {
            interval = 3;
            format = "{usage}% ";
            "on-click" = "kitty -- htop";
          };

          memory = {
            interval = 3;
            format = "{}% ";
            "on-click" = "kitty -- htop";
            "tooltip-format" = "Used: {used:0.1f}G/{total:0.1f}G. Swap: {swapUsed:0.1f}G/{swapTotal:0.1f}G";
            states = {
              critical = 80;
            };
          };

          temperature = {
            interval = 3;
            "hwmon-path" = "/sys/class/hwmon/hwmon1/temp1_input";
            "critical-threshold" = 90;
            "format-critical" = "{temperatureC}°C {icon}";
            format = "{temperatureC}°C {icon}";
            "format-icons" = [
              ""
              ""
              ""
            ];
          };

          "disk#ssd" = {

            interval = 60;
            format = "{path} {free} ";
            path = "/";
            tooltip = true;
            warning = 80;
            critical = 90;
          };

          network = {
            interval = 60;
            on-click = "foot --app-id nmtui nmtui";
            "interface-ethernet" = "enp1s*";
            "interface-wifi" = "wlan0";
            "format-ethernet" = "eth ";
            "format-wifi" = "{essid} ({signalStrength}%) ";
            "tooltip-format-ethernet" = "{ifname}: {ipaddr}/{cidr} ";
            "tooltip-format-wifi" = "{ifname}: {ipaddr}/{cidr} ";
            "format-linked" = "(No IP) ";
            "format-disconnected" = "Disconnected ⚠";
          };

          "custom/power" = {
            "format" = "";
            "on-click" = "wlogout";
            justify = "center";
          };

          "custom/kernel" = {
            exec = "uname -r | sed -E 's/^([0-9]+\\.[0-9]+\\.[0-9]+)-.*-([a-zA-Z0-9]+)/\\1-\\2/'";
            format = "{} ";
          };
          "battery" = {
            "interval" = 30;
            "states" = {
              "good" = 95;
              "warning" = 30;
              "critical" = 20;
            };
            "format" = "{capacity}% {icon}";
            "format-charging" = "{capacity}% 󰂄";
            "format-plugged" = "{capacity}% 󰂄 ";
            "format-alt" = "{time} {icon}";
            "format-icons" = [
              "󰁻"
              "󰁼"
              "󰁾"
              "󰂀"
              "󰂂"
              "󰁹"
            ];
          };

          pulseaudio = {
            "scroll-step" = 2;
            format = "{volume}% {icon}  {format_source}";
            "format-bluetooth" = "{volume}% {icon} {format_source}";
            "format-bluetooth-muted" = "  {icon} {format_source}";
            "format-muted" = "  {format_source}";
            "format-source" = "{volume}% ";
            "format-source-muted" = " ";
            "format-icons" = {
              headphone = " ";
              "hands-free" = " ";
              headset = " ";
              phone = "";
              portable = "";
              car = "";
            };
            "on-click" = "foot --app-id pulsemixer pulsemixer";
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
              border-radius: 1px;
              font-family: JetBrainsMono Nerd Font Propo, Font Awesome;
              font-size: 13px;
              min-height: 0;
          }

          window#waybar {
              background-color: @mantle;
              color: @text;
          }

          window#waybar.hidden {
              opacity: 0.2;
          }

          tooltip {
              background-color: @base;
              border: 1px solid @surface1;
          }

          tooltip label {
              color: @text;
          }

          button {
              box-shadow: inset 0 -3px transparent;
              border: none;
              border-radius: 1;
          }

          button:hover {
              background: inherit;
              box-shadow: inset 0 -3px @text;
          }

          #workspaces button {
              padding: 0 0;
              background-color: @mantle;
              color: @text;
          }

          #workspaces button:hover {
              box-shadow: inherit;
              text-shadow: inherit;
              background-image: linear-gradient(0deg, @surface1, @mantle);
          }

          #workspaces button.active {
              /*background-image: linear-gradient(0deg, @mantle, @surface1);*/
              color: @green;
          }

          #workspaces button.urgent {
              /*background-image: linear-gradient(0deg, @mantle, @mantle);*/
              color: @red;
          }

          #taskbar button.active {
              background-image: linear-gradient(0deg, @surface1, @mantle);
          }

          #mode {
              background-color: @base;
              box-shadow: inset 0 -2px @text;
          }

          #mpris,
          #custom-weather,
          #clock,
          #language,
          #pulseaudio,
          #bluetooth,
          #network,
          #memory,
          #cpu,
          #temperature,
          #disk,
          #custom-kernel,
          #custom-power,
          #idle_inhibitor,
          #scratchpad,
          #battery,
          #mode,
          #tray {
              padding: 0 10px;
              margin: 5px 1px;
              color: @text;
          }

          #window,
          #workspaces {
              margin: 0 4px;
          }

          .modules-left > widget:first-child > #workspaces {
              margin-left: 0;
          }

          .modules-right > widget:last-child > #workspaces {
              margin-right: 0;
          }

          #custom-weather {
              background-color: @teal;
              color: @mantle;
              margin-right: 5px;
          }

          #custom-kernel {
              background-color: @rosewater;
              color: @mantle;
          }

          #custom-power {
              background-color: @peach;
              color: @mantle;
          }

          #clock {
              background-color: @green;
              color: @mantle;
          }

          #battery {
              background-color: @blue;
              color: @mantle;
          }


          @keyframes blink {
              to {
                  background-color: @mantle;
                  color: @text;
              }
          }

          label:focus {
              background-color: @mantle;
          }

          #cpu {
              background-color: @mauve;
              color: @mantle;
              min-width: 45px;
          }

          #memory {
              background-color: @red;
              color: @mantle;
          }

          #disk {
              background-color: @flamingo;
              color: @mantle;
          }

          #network {
              background-color: @peach;
              color: @mantle;
          }

          #network.disconnected {
              background-color: red;
              color: @mantle;
          }

          #bluetooth {
              background-color: @maroon;
              color: @mantle;
              min-width: 40px;
          }

          #pulseaudio {
              background-color: @yellow;
              color: @mantle;
          }

          #pulseaudio.muted {
              background-color: red;
              color: @mantle;
          }

          #temperature {
              background-color: @pink;
              color: @mantle;
              min-width: 37px;
          }

          #temperature.critical {
              background-color: red;
              color: @mantle;
              min-width: 37px;
          }

          #mpris {
              background-color: @base;
              color: @text;
          }

          #tray {
              background-color: @overlay0;
              color: @text;
          }

          #tray > .passive {
              -gtk-icon-effect: dim;
          }

          #tray > .needs-attention {
              -gtk-icon-effect: highlight;
              background-color: @mantle;
          }

          #idle_inhibitor {
              background-color: @base;
              color: @text;
              font-family: Inter;
          }

          #idle_inhibitor.activated {
              background-color: @text;
              color: @base;
          }

          #scratchpad {
              background-color: @base;
              color: @text;
          }

          #scratchpad.empty {
              background-color: transparent;
          }
      '';
    };
  };
}
