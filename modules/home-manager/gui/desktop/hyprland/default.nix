{
  lib,
  pkgs,
  config,
  ...
}:
{
  imports = [
    ./waybar.nix
    ./wlogout
    ./hyprlock.nix
    ./hyprpaper
  ];
  options = {
    gui-settings.desktop.hyprland.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.gui-settings.desktop.enable;
      description = "Enable hyprland for desktop environment";
    };
  };
  config = lib.mkIf config.gui-settings.desktop.hyprland.enable {
    # Packages required by hyprland
    home.packages = with pkgs; [
      nautilus # File manager
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
        "$FILE" = "nautilus";
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
            # binds $mod + [shift +] {1..9} to [move to] workspace {1..9}
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
              ) 9
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
          "float,title:^(fly_is_foot)$"
          "move center,title:^(fly_is_foot)$"
          "float,title:^(Spotify)$"
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
          disable_hyprland_logo = false;
        };
        gestures = {
          workspace_swipe = true;
        };
        xwayland = {
          force_zero_scaling = true;
        };
        exec-once = [
          "fcitx5 -d --replace"
          # "waybar"
          "dunst"
        ];
        env = [
          "XDG_CURRENT_DESKTOP, Hyprland"
          "XDG_SESSION_TYPE, wayland"
          "XDG_SESSION_DESKTOP, Hyprland"
          "GDK_SCALE, 2"
          "XCURSOR_SIZE, 32"

          "EDITOR, nvim"
          "MOZ_ENABLE_WAYLAND, 1"
        ];
      };
    };
    # Notification daemon:
    services.dunst = {
      enable = true;
    };
    # Windows switcher / App launcher:
    programs.rofi = {
      enable = true;
    };
    # XDG portal:
    xdg.portal.enable = true;
    xdg.portal.extraPortals = [ pkgs.xdg-desktop-portal-gtk ];
    # Hint Electron apps to use Wayland:
    home.sessionVariables = {
      WLR_NO_HARDWARE_CURSORS = "1";
      NIXOS_OZONE_WL = "1";
    };
  };
}
