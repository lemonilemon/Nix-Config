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
        pkgs.coreutils
      ]
    }
    for _ in $(seq 1 50); do
      if awww query >/dev/null 2>&1; then
        break
      fi
      sleep 0.1
    done
    # The daemon runs --no-cache, so every start is a blank slate and this
    # script owns the reveal: mocha base first, then the wallpaper grows in
    # from the top right -- the same transition an in-session pick uses
    # (transition values match awwwTransition in
    # eww/go/internal/collect/wallpaperctl.go, which also writes the state
    # file read here).
    current="$(cat "$HOME/.local/state/awww/current-wallpaper" 2>/dev/null || true)"
    if [ ! -f "$current" ]; then
      current="$HOME/Pictures/wallpapers/pixel_sunset.png"
    fi
    awww clear 1e1e2e
    awww img "$current" \
      --transition-type grow --transition-pos top-right \
      --transition-duration 0.8 --transition-fps 60
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
        # --no-cache: the reveal in awww-init is the only path that draws at
        # startup; the daemon restoring its cache first would race it with an
        # untransitioned frame.
        ExecStart = "${pkgs.awww}/bin/awww-daemon --no-cache";
        ExecStartPost = awwwInit;
        MemoryAccounting = true;
        Restart = "on-failure";
        RestartSec = "1s";
      };

      Install.WantedBy = [ "hyprland-session.target" ];
    };
  };
}
