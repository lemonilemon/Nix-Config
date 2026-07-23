{
  lib,
  config,
  pkgs,
  ...
}:
let
  cfg = config.home.desktop.hyprland;

  # Repo source of truth stays JXL (hyprpaper still uses it); swww cannot
  # decode JXL, so derive a PNG seed at build time.
  seedPng = pkgs.runCommand "pixel-sunset-png" { nativeBuildInputs = [ pkgs.imagemagick ]; } ''
    magick ${../hyprpaper/wallpaper/pixel_sunset.jxl} png:$out
  '';

  swwwInit = pkgs.writeShellScript "swww-init" ''
    export PATH=${
      lib.makeBinPath [
        pkgs.swww
        pkgs.gnugrep
        pkgs.coreutils
      ]
    }
    for _ in $(seq 1 50); do
      if swww query >/dev/null 2>&1; then
        break
      fi
      sleep 0.1
    done
    # swww restores its own per-output cache on start; only seed a blank slate.
    if ! swww query 2>/dev/null | grep -q "image: "; then
      swww img "$HOME/Pictures/wallpapers/pixel_sunset.png" --transition-type none
    fi
  '';
in
{
  config = lib.mkIf cfg.swww.enable {
    home.packages = [ pkgs.swww ];

    home.file."Pictures/wallpapers/pixel_sunset.png".source = seedPng;

    systemd.user.services.swww-daemon = {
      Unit = {
        Description = "swww wallpaper daemon";
        After = [ "hyprland-session.target" ];
        PartOf = [ "hyprland-session.target" ];
      };

      Service = {
        ExecStart = "${pkgs.swww}/bin/swww-daemon";
        ExecStartPost = swwwInit;
        Restart = "on-failure";
        RestartSec = "1s";
      };

      Install.WantedBy = [ "hyprland-session.target" ];
    };
  };
}
