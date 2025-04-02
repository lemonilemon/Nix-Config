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
    ./network
    ./wlogout
    ./hyprpaper
  ];
  config = lib.mkIf config.home.desktop.hyprland.enable {
    # Packages required by hyprland
    home.packages = with pkgs; [
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
      settings = {
        "$MOD1" = "SUPER";
        "$MOD2" = "alt";
        "$TERM" = "kitty";
        "$DRUN" = "rofi -show drun";
        "$BROWSER" = "firefox";
        "$FILE" = "nemo";
        bind =
          [
            "$MOD1, m, exit,"
            "$MOD1, x, killactive,"
            "$MOD1, f, fullscreen,"
            "$MOD1, s, togglefloating,"
            "$MOD1, up, movefocus, u"
            "$MOD1, down, movefocus, d"
            "$MOD1, left, movefocus, l"
            "$MOD1, right, movefocus, r"
            "$MOD1, g, togglegroup,"
            "$MOD1, tab, changegroupactive,"
            "$MOD1, c, centerwindow,"
            "$MOD1, Tab, cyclenext," # change focus to another window

            "$MOD2, f, exec, $BROWSER" # Browser
            "$MOD2, q, exec, $TERM" # Terminal
            "$MOD2, space, exec, $DRUN" # Launcher
            "$MOD2, e, exec, $FILE" # File manager

            ", Print, exec, grim -g \"$(slurp)\" - | wl-copy"
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
                  "$MOD1, code:1${toString i}, workspace, ${toString ws}"
                  "$MOD1 SHIFT, code:1${toString i}, movetoworkspace, ${toString ws}"
                ]
              ) 5
            )
          );

        bindm = [
          "$MOD1,mouse:272,movewindow"
          "$MOD1,mouse:273,resizewindow"
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
        ];

        input = {
          touchpad = {
            natural_scroll = true;
            tap-and-drag = true;
            tap-to-click = true;
          };
        };
        misc = {
          force_default_wallpaper = "-1";
          disable_hyprland_logo = true;
        };
        gestures = {
          workspace_swipe = true;
        };
        xwayland = {
          force_zero_scaling = true;
        };
        exec-once = [
          "waybar"
          "nm-applet --indicator"
          "fcitx5 -d --replace"
          "dunst"
        ];
        cursor = {
          no_hardware_cursors = true;
        };
        env = [
          "XDG_CURRENT_DESKTOP, Hyprland"
          "XDG_SESSION_TYPE, wayland"
          "XDG_SESSION_DESKTOP, Hyprland"
          "ELECTRON_OZONE_PLATFORM_HINT,wayland"
          "GDK_SCALE, 1.15"
          "XCURSOR_SIZE, 32"

          # "GDK_BACKEND, \"wayland,x11\""
          "QT_QPA_PLATFORM, wayland"
          "QT_QPA_PLATFORMTHEME, qt5ct"
          # "QT_AUTO_SCREEN_SCALE_FACTOR,1"
          # "QT_SCALE_FACTOR, 1.15"
          # "GDK_DPI_SCALE, 1.15"

          # fcitx input-related
          "GLFW_IM_MODULE, fcitx"
          # "GTK_IM_MODULE, fcitx" # recommended to disable
          "QT_IM_MODULE, fcitx"
          "INPUT_METHOD, fcitx"
          "XMODIFIERS, @im=fcitx"
          "IMSETTINGS_MODULE, fcitx"

          "EDITOR, nvim"
          "MOZ_ENABLE_WAYLAND, 1"
          "MOZ_WEBRENDER, 1"
        ];
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
      font = "JetBrains Mono Nerd Font 14";
      extraConfig = {
        show-icons = true;
        modi = "window,drun,run,ssh";
        combi-modi = "drun,run,ssh";
        matching = "fuzzy";
        terminal = "kitty";
        fake-transparency = true;
        sidebar-mode = true;
      };
    };

    # XDG portal:
    xdg.portal.enable = true;
    xdg.portal.extraPortals = [ pkgs.xdg-desktop-portal-gtk ];

    # Hint Electron apps to use Wayland:
    home.sessionVariables = {
      WLR_NO_HARDWARE_CURSORS = "1";
      NIXOS_OZONE_WL = "1";
      # QT4_IM_MODULE = "fcitx";
      # QT_IM_MODULE = "fcitx";
      # QT_QPA_PLATFORM = "wayland";
      # "GTK_IM_MODULE, fcitx"
    };
  };
}
