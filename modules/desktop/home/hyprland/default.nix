{
  lib,
  pkgs,
  config,
  ...
}:
{
  imports = [
    ./eww
    ./waybar.nix
    ./dunst.nix
    ./hyprlock.nix
    ./nemo.nix
    ./imv.nix
    ./rofi.nix
    ./theme
    ./network
    ./wlogout
    ./hyprpaper
    ./wallpaper
    ./bluetooth
  ];
  config = lib.mkIf config.home.desktop.hyprland.enable {
    # Packages required by hyprland
    home.packages = with pkgs; [
      hyprpolkitagent # Polkit agent
      grim # Grab images
      slurp # Select region
      wl-clipboard # Clipboard manager
      brightnessctl # Brightness control
      playerctl # Player control
      pw-volume # Volume control
      nwg-displays # Monitor layout GUI
      hyprshutdown # Graceful Hyprland shutdown
    ];

    wayland.windowManager.hyprland = {
      enable = true; # enable Hyprland
      xwayland.enable = true; # enable xwayland
      systemd.enable = true; # import env vars into systemd user session
      # Hyprland 0.57 drops the hyprlang .conf format; settings render to
      # ~/.config/hypr/hyprland.lua as hl.*() calls.
      configType = "lua";
      settings =
        let
          inherit (lib.generators) mkLuaInline;
          toLua = lib.generators.toLua { };
          # hl.dsp dispatcher call as a raw Lua expression
          dsp = expr: mkLuaInline "hl.dsp.${expr}";
          exec = cmd: dsp "exec_cmd(${toLua cmd})";
          bind = key: action: {
            _args = [
              key
              action
            ];
          };
          bindWith = key: action: flags: {
            _args = [
              key
              action
              flags
            ];
          };
          MOD1 = "SUPER";
          MOD2 = "ALT + SHIFT";
          MOD3 = "ALT";
          MOD4 = "SUPER + ALT";
          TERM = "kitty";
          DRUN = "rofi -show drun";
          BROWSER = "zen-beta";
          FILE = "nemo";
        in
        {
          bind = [
            (bind "${MOD1} + M" (dsp "exit()"))
            (bind "${MOD1} + X" (dsp "window.close()"))
            (bind "${MOD1} + F" (dsp "window.fullscreen()"))
            (bind "${MOD1} + T" (dsp ''window.float({ action = "toggle" })''))
            # For window movement
            (bind "${MOD4} + S" (dsp ''window.move({ workspace = "special:scratchpad" })''))
            # Vim-like keybindings for focus movement
            (bind "${MOD1} + K" (dsp ''focus({ direction = "up" })''))
            (bind "${MOD1} + J" (dsp ''focus({ direction = "down" })''))
            (bind "${MOD1} + H" (dsp ''focus({ direction = "left" })''))
            (bind "${MOD1} + L" (dsp ''focus({ direction = "right" })''))
            (bind "${MOD3} + Tab" (dsp "window.cycle_next()")) # change focus to another window

            (bind "${MOD2} + T" (exec TERM)) # New terminal (non-special workspace)
            (bind "Print" (exec ''grim -g "$(slurp)" - | wl-copy''))
            (bind "${MOD1} + SHIFT + S" (exec ''grim -g "$(slurp)" - | wl-copy''))

            # Release binds: trigger on key release
            (bindWith "${MOD2} + F" (exec BROWSER) { release = true; }) # Browser
            (bindWith "${MOD2} + Q" (dsp ''workspace.toggle_special("mainterm")'') {
              release = true;
            }) # Terminal
            (bindWith "${MOD2} + S" (dsp ''workspace.toggle_special("scratchpad")'') {
              release = true;
            }) # Scratchpad
            # Toggle: `pkill rofi` closes an open launcher; only if none was
            # running does the right-hand side start one.
            (bindWith "${MOD2} + space" (exec "pkill rofi || ${DRUN}") { release = true; })
            (bindWith "${MOD2} + E" (exec FILE) { release = true; }) # File manager

            (bindWith "${MOD1} + mouse:272" (dsp "window.drag()") { mouse = true; })
            (bindWith "${MOD1} + mouse:273" (dsp "window.resize()") { mouse = true; })

            # Media keys: repeat while held, work on the lock screen
            (bindWith "XF86AudioRaiseVolume" (exec "wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%+") {
              locked = true;
              repeating = true;
            })
            (bindWith "XF86AudioLowerVolume" (exec "wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-") {
              locked = true;
              repeating = true;
            })
            (bindWith "XF86MonBrightnessDown" (exec "brightnessctl set 5%-") {
              locked = true;
              repeating = true;
            })
            (bindWith "XF86MonBrightnessUp" (exec "brightnessctl set +5%") {
              locked = true;
              repeating = true;
            })
            (bindWith "XF86AudioMute" (exec "wpctl set-mute @DEFAULT_AUDIO_SINK@ toggle") {
              locked = true;
            })
            (bindWith "XF86AudioPlay" (exec "playerctl play-pause") { locked = true; })
            (bindWith "XF86AudioPrev" (exec "playerctl previous") { locked = true; })
            (bindWith "XF86AudioNext" (exec "playerctl next") { locked = true; })
          ]
          ++
            lib.optionals
              (config.home.desktop.hyprland.eww.enable && config.home.desktop.hyprland.eww.laptopControls.enable)
              [
                (bind "${MOD1} + SHIFT + O" (exec "eww-barctl display toggle"))
              ]
          ++ lib.optionals config.home.desktop.hyprland.eww.enable [
            (bind "${MOD1} + N" (
              exec "eww-popup toggle notif_center_popup && eww-barctl --quiet notif mark-seen"
            ))
            (bind "${MOD1} + CTRL + N" (exec "eww-barctl --quiet notif dnd-toggle"))
            (bind "${MOD1} + W" (
              exec "eww-barctl --quiet wallpaper rescan && eww-popup toggle wallpaper_picker_popup"
            ))
          ]
          ++ lib.optionals config.home.desktop.hyprland.dunst.enable [
            (bind "${MOD1} + Escape" (exec "dunstctl close"))
            (bind "${MOD1} + SHIFT + Escape" (exec "dunstctl close-all"))
          ]
          ++ (
            # workspaces
            # binds $mod + [shift +] {1..5} to [move to] workspace {1..5}
            builtins.concatLists (
              builtins.genList (
                i:
                let
                  ws = toString (i + 1);
                in
                [
                  (bind "${MOD1} + code:1${toString i}" (dsp "focus({ workspace = ${ws} })"))
                  (bind "${MOD1} + SHIFT + code:1${toString i}" (dsp "window.move({ workspace = ${ws} })"))
                ]
              ) 5
            )
          );

          window_rule = [
            {
              match.class = BROWSER;
              tag = "+browser";
            }
            {
              match.class = TERM;
              tag = "+term";
            }
            {
              match.class = FILE;
              tag = "+file";
            }
            {
              match.modal = true;
              pseudo = true;
            }
            {
              match.class = "fcitx5";
              pseudo = true;
            }
            {
              match = {
                tag = "term";
                workspace = "special:mainterm";
              };
              tag = "+mainterm";
            }
            {
              match.tag = "(browser|file)";
              opacity = 0.8;
            }
          ];

          layer_rule =
            lib.optionals config.home.desktop.hyprland.waybar.enable [
              {
                match.namespace = "waybar";
                blur = true;
              }
            ]
            ++ lib.optionals config.home.desktop.hyprland.eww.enable (
              map
                (ns: {
                  match.namespace = ns;
                  blur = true;
                })
                [
                  "eww-bar"
                  "eww-volume"
                  "eww-display-mode"
                  "eww-bluetooth"
                  "eww-network"
                  "eww-battery"
                  "eww-ai-usage"
                  "eww-notifications"
                  "eww-wallpaper"
                  "eww-settings"
                ]
            )
            ++ lib.optionals config.home.desktop.hyprland.dunst.enable [
              {
                match.namespace = "notifications";
                blur = true;
              }
            ]
            ++ lib.optionals config.home.desktop.hyprland.wlogout.enable [
              # wlogout names its layer surface logout_dialog. Without this it is
              # the only layer surface on the desktop left with a razor-sharp
              # backdrop, which is most of why the power menu read as foreign
              # beside hyprlock and its blur_passes = 3.
              {
                match.namespace = "logout_dialog";
                blur = true;
              }
            ]
            ++ [
              # rofi 2.0 is Wayland-native and runs as a layer surface named
              # `rofi`, so this -- not a windowrule -- is what reaches it. Blur
              # completes the launcher's glass panel (see ./rofi.nix), the same
              # treatment logout_dialog gets.
              {
                match.namespace = "rofi";
                blur = true;
              }
            ];

          workspace_rule = [
            {
              workspace = "special:mainterm";
              on_created_empty = TERM;
            }
          ];

          config = {
            input = {
              touchpad = {
                natural_scroll = true;
                tap_and_drag = true;
                tap_to_click = true;
              };
              accel_profile = "adaptive";
              sensitivity = 0;
            };
            xwayland = {
              force_zero_scaling = true;
            };
            cursor = {
              no_hardware_cursors = true;
            };
            decoration = {
              rounding = 16;
              blur = {
                size = 10;
                passes = 1;
              };
            };
            misc = {
              force_default_wallpaper = -1;
              disable_hyprland_logo = true;
              # Mocha base for the frame between login and awww's first draw;
              # the default (0xff111111) is the one off-palette surface in the
              # whole startup chain.
              background_color = mkLuaInline "0xff1e1e2e";
            };
            ecosystem = {
              no_update_news = true;
              no_donation_nag = true;
            };
          };
          device = {
            name = "logitech-g-pro--1";
            sensitivity = -0.2;
          };
          gesture = [
            {
              fingers = 3;
              direction = "horizontal";
              action = "workspace";
            }
            {
              fingers = 3;
              direction = "down";
              mods = "ALT";
              action = "close";
            }
            {
              fingers = 3;
              direction = "up";
              mods = "SUPER";
              action = "fullscreen";
            }
          ];
          # Open when hyprland starts
          on = [
            {
              _args = [
                "hyprland.start"
                (mkLuaInline ''
                  function()
                    hl.exec_cmd("fcitx5 -d --replace")
                    -- polkit
                    hl.exec_cmd("systemctl --user start hyprpolkitagent")
                    -- discord
                    hl.exec_cmd("webcord --start-minimized")
                    -- 1password
                    hl.exec_cmd("1password --silent")
                    hl.exec_cmd("blueman-applet")
                  end'')
              ];
            }
          ];
          env = [
            {
              _args = [
                "ELECTRON_OZONE_PLATFORM_HINT"
                "wayland"
              ];
            }
            {
              _args = [
                "NIXOS_OZONE_WL"
                "1"
              ];
            }
            {
              _args = [
                "NVD_BACKEND"
                "direct"
              ];
            }
            {
              _args = [
                "XDG_CURRENT_DESKTOP"
                "Hyprland"
              ];
            }
            {
              _args = [
                "XDG_SESSION_TYPE"
                "wayland"
              ];
            }
            {
              _args = [
                "XDG_SESSION_DESKTOP"
                "Hyprland"
              ];
            }

            {
              _args = [
                "GDK_SCALE"
                "1.15"
              ];
            }
            {
              _args = [
                "GDK_BACKEND"
                "wayland"
              ];
            }
            {
              _args = [
                "QT_QPA_PLATFORM"
                "wayland"
              ];
            }
            {
              _args = [
                "QT_QPA_PLATFORMTHEME"
                "qt5ct"
              ];
            }
            {
              _args = [
                "QT_AUTO_SCREEN_SCALE_FACTOR"
                "1"
              ];
            }
            {
              _args = [
                "QT_SCALE_FACTOR"
                "1.15"
              ];
            }
            {
              _args = [
                "GDK_DPI_SCALE"
                "1.15"
              ];
            }

            # fcitx input-related
            {
              _args = [
                "GLFW_IM_MODULE"
                "fcitx"
              ];
            }
            {
              _args = [
                "QT_IM_MODULE"
                "wayland;fcitx;ibus"
              ];
            }
            {
              _args = [
                "XMODIFIERS"
                "@im=fcitx"
              ];
            }

            {
              _args = [
                "EDITOR"
                "nvim"
              ];
            }
            {
              _args = [
                "MOZ_ENABLE_WAYLAND"
                "1"
              ];
            }
            {
              _args = [
                "MOZ_WEBRENDER"
                "1"
              ];
            }
          ];
          # Fallback: overridden per-output by monitors.lua when it exists
          monitor = {
            output = "";
            mode = "preferred";
            position = "auto";
            scale = 1.0;
          };
        };
      extraConfig = ''
        -- nwg-displays writes per-output overrides here; absent on a fresh
        -- install, hence the pcall.
        pcall(dofile, os.getenv("HOME") .. "/.config/hypr/monitors.lua")
      '';
    };

    home.sessionVariables = {
      WLR_NO_HARDWARE_CURSORS = "1";
    };
  };
}
