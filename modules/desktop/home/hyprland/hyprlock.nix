{
  lib,
  config,
  catppuccin,
  ...
}:
let
  # Minecraft-style splash messages — add your own here!
  splashMessages = [
    "Also try NixOS!"
    "Declarative all the way down!"
    "Have you run nixos-rebuild switch today?"
    "Works on my machine... because it's reproducible!"
    "There is no place like 127.0.0.1"
    "It's not a bug, it's a feature!"
    "sudo make me a sandwich"
    "rm -rf /tmp/bad-ideas"
    "Git gud at Git!"
    "Functional programming enjoyer!"
    "Wayland native!"
    "Keep calm and nix flake update"
    "The real friends were the derivations we made along the way"
    "Ricing in progress..."
    "100% reproducible!"
  ];
  numSplash = builtins.length splashMessages;
  splashArray = lib.concatMapStringsSep " " (m: "\"${m}\"") splashMessages;
in
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    catppuccin.hyprlock.enable = false;
    programs.hyprlock = {
      enable = true;
      settings = {
        general = {
          disable_loading_bar = true;
          hide_cursor = true;
        };
        background = [
          {
            monitor = "";
            path = "screenshot";
            blur_passes = 3;
            blur_size = 7;
            noise = 0.012;
            contrast = 0.9;
            brightness = 0.8;
            vibrancy = 0.17;
          }
        ];
        label = [
          # Clock
          {
            monitor = "";
            text = ''cmd[update:1000] echo "$(date +"%-H:%M")"'';
            color = "rgba(a6e3a1ff)";
            font_size = 48;
            font_family = "JetBrains Mono Nerd Font Bold";
            position = "0, 65";
            halign = "center";
            valign = "center";
          }
          # Date
          {
            monitor = "";
            text = ''cmd[update:60000] echo "$(date +"%A, %d %B %Y")"'';
            color = "rgba(bac2deff)";
            font_size = 14;
            font_family = "JetBrains Mono Nerd Font";
            position = "0, 18";
            halign = "center";
            valign = "center";
          }
          # Splash message
          {
            monitor = "";
            text = ''cmd[update:0] msgs=(${splashArray}); echo "''${msgs[$((RANDOM % ${toString numSplash}))]}"'';
            color = "rgba(f9e2afff)";
            font_size = 14;
            font_family = "JetBrains Mono Nerd Font Italic";
            position = "0, -60";
            halign = "center";
            valign = "top";
          }
        ];
        "input-field" = [
          {
            monitor = "";
            size = "280, 48";
            outline_thickness = 2;
            dots_size = 0.2;
            dots_spacing = 0.2;
            dots_center = true;
            outer_color = "rgba(89b4fa80)";
            inner_color = "rgba(313244cc)";
            font_color = "rgba(cdd6f4ff)";
            fade_on_empty = true;
            placeholder_text = "󰌾  Enter Password";
            rounding = 12;
            check_color = "rgba(a6e3a1ff)";
            fail_color = "rgba(f38ba8ff)";
            fail_text = "<i>Wrong password ($ATTEMPTS)</i>";
            position = "0, -70";
            halign = "center";
            valign = "center";
          }
        ];
      };
    };
    services.hypridle = {
      enable = true;
      settings = {
        general = {
          lock_cmd = "pidof hyprlock || hyprlock"; # avoid starting multiple hyprlock instances.
          before_sleep_cmd = "loginctl lock-session"; # lock before suspend.
          # Re-enable display then restart hyprlock if it crashed during GPU reset on suspend.
          after_sleep_cmd = "hyprctl dispatch dpms on; pidof hyprlock || hyprlock";
        };
        listener = [
          {
            timeout = 900; # 15 min
            on-timeout = "loginctl lock-session";
          }
          {
            # Turn off display while locked to save power.
            # on-resume brings it back if user wakes before suspend fires.
            timeout = 1200; # 20 min
            on-timeout = "hyprctl dispatch dpms off";
            on-resume = "hyprctl dispatch dpms on";
          }
          {
            timeout = 1800; # 30 min
            on-timeout = "systemctl suspend";
          }
        ];
      };
    };
  };
}
