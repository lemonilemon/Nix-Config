{
  lib,
  config,
  pkgs,
  ...
}:
let
  cfg = config.home.desktop.hyprland;

  # Repo source of truth stays JXL (hyprpaper still uses it); awww cannot
  # decode JXL, so derive a PNG seed at build time.
  seedPng = pkgs.runCommand "pixel-sunset-png" { nativeBuildInputs = [ pkgs.imagemagick ]; } ''
    magick ${../hyprpaper/wallpaper/pixel_sunset.jxl} png:$out
  '';

  awwwInit = pkgs.writeShellScript "awww-init" ''
    export PATH=${
      lib.makeBinPath [
        pkgs.awww
        pkgs.gnugrep
        pkgs.coreutils
      ]
    }
    for _ in $(seq 1 50); do
      if awww query >/dev/null 2>&1; then
        break
      fi
      sleep 0.1
    done
    # awww restores its own per-output cache on start; only seed a blank slate.
    if ! awww query 2>/dev/null | grep -q "image: "; then
      awww img "$HOME/Pictures/wallpapers/pixel_sunset.png" --transition-type none
    fi
  '';
in
{
  config = lib.mkIf cfg.awww.enable {
    home.packages = [ pkgs.awww ];

    home.file."Pictures/wallpapers/pixel_sunset.png".source = seedPng;

    systemd.user.services.awww-daemon = {
      Unit = {
        Description = "awww wallpaper daemon";
        After = [ "hyprland-session.target" ];
        PartOf = [ "hyprland-session.target" ];
      };

      Service = {
        ExecStart = "${pkgs.awww}/bin/awww-daemon";
        ExecStartPost = awwwInit;
        Restart = "on-failure";
        RestartSec = "1s";
      };

      Install.WantedBy = [ "hyprland-session.target" ];
    };
  };
}
