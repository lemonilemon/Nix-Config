{
  lib,
  pkgs,
  config,
  ...
}:
{
  imports = [
    ./waybar.nix
    ./hyprlock.nix
    ./ranger.nix
    ./nemo.nix
    ./imv.nix
    ./theme
    ./network
    ./wlogout
    ./hyprpaper
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
    ];

    wayland.windowManager.hyprland = {
      enable = true; # enable Hyprland
      xwayland.enable = true; # enable xwayland
      settings =
        let
          MOD1 = "SUPER";
          MOD2 = "alt+shift";
          MOD3 = "alt";
          MOD4 = "SUPER+alt";
          TERM = "kitty";
          DRUN = "rofi -show drun";
          BROWSER = "zen-beta";
          FILE = "nemo";
        in
        {
          bind = [
            "${MOD1}, m, exit,"
            "${MOD1}, x, killactive,"
            "${MOD1}, f, fullscreen,"
            "${MOD1}, t, togglefloating,"
            # For window movement
            "${MOD4}, s, movetoworkspace, special:scratchpad"
            "${MOD4}, 1, movetoworkspace, 1"
            "${MOD4}, 2, movetoworkspace, 2"
            "${MOD4}, 3, movetoworkspace, 3"
            "${MOD4}, 4, movetoworkspace, 4"
            "${MOD4}, 5, movetoworkspace, 5"
            # Vim-like keybindings for focus movement
            "${MOD1}, k, movefocus, u"
            "${MOD1}, j, movefocus, d"
            "${MOD1}, h, movefocus, l"
            "${MOD1}, l, movefocus, r"
            "${MOD3}, Tab, cyclenext," # change focus to another window

            ", Print, exec, grim -g \"$(slurp)\" - | wl-copy"
            "${MOD1}+SHIFT, s, exec, grim -g \"$(slurp)\" - | wl-copy"
          ]
          ++ (
            # workspaces
            # binds $mod + [shift +] {1..5} to [move to] workspace {1..5}
            builtins.concatLists (
              builtins.genList (
                i:
                let
                  ws = i + 1;
                in
                [
                  "${MOD1}, code:1${toString i}, workspace, ${toString ws}"
                  "${MOD1}+SHIFT, code:1${toString i}, movetoworkspace, ${toString ws}"
                ]
              ) 5
            )
          );
          bindr = [
            "${MOD2}, f, exec, ${BROWSER}" # Browser
            "${MOD2}, q, togglespecialworkspace, mainterm" # Terminal
            "${MOD2}, s, togglespecialworkspace, scratchpad" # Scratchpad
            "${MOD2}, space, exec, pkill ${DRUN} || ${DRUN}" # Launcher
            "${MOD2}, e, exec, ${FILE}" # File manager
          ];

          bindm = [
            "${MOD1},mouse:272,movewindow"
            "${MOD1},mouse:273,resizewindow"
          ];
          bindel = [
            ", XF86AudioRaiseVolume, exec, wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%+"
            ", XF86AudioLowerVolume, exec, wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-"
            ", XF86MonBrightnessDown, exec, brightnessctl set 5%-"
            ", XF86MonBrightnessUp, exec, brightnessctl set +5%"
          ];
          bindl = [
            ", XF86AudioMute, exec, wpctl set-mute @DEFAULT_AUDIO_SINK@ toggle"
            # Requires playerctl
            ", XF86AudioPlay, exec, playerctl play-pause"
            ", XF86AudioPrev, exec, playerctl previous"
            ", XF86AudioNext, exec, playerctl next"
          ];

          monitor = [
            ",preferred,auto,1"
          ];

          # rule
          windowrule = [
            "tag +browser, class:(${BROWSER})$"
            "tag +term, class:(${TERM})$"
            "tag +file, class:(${FILE})$"
            "tag +launcher, class:(rofi|Rofi)$"
            "stayfocused, tag:launcher"
            "pseudo, class:(fcitx5)$"
            "maximize, tag:term, onworkspace:special:mainterm"
            "tag +mainterm, tag:term, onworkspace:special:mainterm"
            "opacity 0.8, tag:(browser|file)"
          ];

          workspace = [
            "special:mainterm, on-created-empty: ${TERM}"
          ];

          input = {
            touchpad = {
              natural_scroll = true;
              tap-and-drag = true;
              tap-to-click = true;
            };
            accel_profile = "adaptive";
            sensitivity = 0;
          };
          device = [
            {
              # Logitech G Pro Wireless
              name = "logitech-g-pro--1";
              sensitivity = -0.2;
            }
          ];
          gesture = [
            "3, horizontal, workspace"
            "3, down, mod: ALT, close"
            "3, up, mod: SUPER, fullscreen"
          ];
          xwayland = {
            force_zero_scaling = true;
          };
          cursor = {
            no_hardware_cursors = true;
          };
          # Open when hyprland starts
          exec-once = [
            "waybar"
            # "nm-applet --indicator"
            "fcitx5 -d --replace"
            "dunst"
            # polkit
            "systemctl --user start hyprpolkitagent"
            # discord
            "webcord --start-minimized"
            # 1password
            "1password --silent"
          ];
          env = [
            "ELECTRON_OZONE_PLATFORM_HINT, wayland"
            "NIXOS_OZONE_WL, 1"
            "NVD_BACKEND,direct"
            "XDG_CURRENT_DESKTOP, Hyprland"
            "XDG_SESSION_TYPE, wayland"
            "XDG_SESSION_DESKTOP, Hyprland"

            "GDK_SCALE, 1.15"
            "GDK_BACKEND, wayland"
            "QT_QPA_PLATFORM, wayland"
            "QT_QPA_PLATFORMTHEME, qt5ct"
            "QT_AUTO_SCREEN_SCALE_FACTOR,1"
            "QT_SCALE_FACTOR, 1.15"
            "GDK_DPI_SCALE, 1.15"

            # fcitx input-related
            "GLFW_IM_MODULE, fcitx"
            "QT_IM_MODULE, wayland;fcitx;ibus"
            "XMODIFIERS, @im=fcitx"

            "EDITOR, nvim"
            "MOZ_ENABLE_WAYLAND, 1"
            "MOZ_WEBRENDER, 1"
          ];
          decoration = {
            rounding = 16;
          };
          misc = {
            force_default_wallpaper = "-1";
            disable_hyprland_logo = true;
          };
          ecosystem = {
            no_update_news = true;
            no_donation_nag = true;
          };
        };
    };
    # Notification daemon:
    services.dunst = {
      enable = true;
      settings = {
        global = {
          frame_color = "#89b4fa";
          separator_color = "frame";
          highlight = "#89b4fa";
          transparency = 10;
          origin = "top-right";
          font = "JetBrains Mono Nerd Font 10";
        };

        urgency_low = {
          background = "#1e1e2e";
          foreground = "#cdd6f4";
        };

        urgency_normal = {
          background = "#1e1e2e";
          foreground = "#cdd6f4";
        };

        urgency_critical = {
          background = "#1e1e2e";
          foreground = "#cdd6f4";
          frame_color = "#fab387";
        };
      };
    };

    # Windows switcher / App launcher:
    programs.rofi = {
      enable = true;
      package = pkgs.rofi;
      font = "JetBrains Mono Nerd Font 14";
      extraConfig = {
        show-icons = true;
        modi = "window,drun,run,ssh";
        combi-modi = "drun,run,ssh";
        matching = "fuzzy";
        terminal = "kitty";
        drun-match-fields = "name";
        drun-dis-play-format = "{name}";
        kb-cancel = "Escape";
        sidebar-mode = true;
      };
    };

    home.sessionVariables = {
      WLR_NO_HARDWARE_CURSORS = "1";
    };
  };
}
