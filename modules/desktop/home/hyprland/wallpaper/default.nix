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

  # Shared by awww-init and awww-monitor-watch: they are the only two paths that
  # draw outside the picker, and all three should reveal identically. These are
  # calibration values matching awwwTransition in
  # eww/go/internal/collect/wallpaperctl.go -- grow-from-top-right is where the
  # picker popup sits.
  transitionArgs = "--transition-type grow --transition-pos top-right --transition-duration 0.8 --transition-fps 60";

  # Sourced by both scripts rather than interpolated into each, so the fallback
  # and the state-file path cannot drift apart.
  awwwCommon = pkgs.writeText "awww-common.sh" ''
    # Sets $current to the wallpaper that should be on screen. The state file is
    # written by PersistCurrentWallpaper in eww/go/internal/collect/wallpaperctl.go;
    # it is the only thing carrying a pick across logins, since the daemon runs
    # --no-cache. Missing or stale means nothing has been picked yet: use the seed.
    resolve_current() {
      current="$(cat "$HOME/.local/state/awww/current-wallpaper" 2>/dev/null || true)"
      if [ ! -f "$current" ]; then
        current="$HOME/Pictures/wallpapers/pixel_sunset.png"
      fi
    }
  '';

  awwwInit = pkgs.writeShellScript "awww-init" ''
    export PATH=${
      lib.makeBinPath [
        pkgs.awww
        pkgs.coreutils
      ]
    }
    . ${awwwCommon}

    for _ in $(seq 1 50); do
      if awww query >/dev/null 2>&1; then
        break
      fi
      sleep 0.1
    done
    # The daemon runs --no-cache, so every start is a blank slate and this
    # script owns the reveal: mocha base first, then the wallpaper grows in
    # from the top right -- the same transition an in-session pick uses.
    resolve_current
    awww clear 1e1e2e
    awww img "$current" ${transitionArgs}
  '';

  # --no-cache buys the clean reveal above, but it also disables the only thing
  # that would paint an output appearing LATER: the daemon's per-output cache
  # restore. A monitor plugged in after login therefore keeps its default black
  # fill forever, because awww-init's `awww img` only ever reached the outputs
  # that existed when it ran. This watcher is the replacement for that restore.
  awwwMonitorWatch = pkgs.writeShellScript "awww-monitor-watch" ''
    export PATH=${
      lib.makeBinPath [
        pkgs.awww
        pkgs.coreutils
        pkgs.gnugrep
        pkgs.gnused
        pkgs.socat
      ]
    }
    . ${awwwCommon}

    # `awww query` prints "<namespace>: <output>: <WxH>, scale: ..., currently
    # displaying: <image|color>: <value>" per output; the namespace is empty
    # here, so every line starts ": ".
    output_names() {
      awww query 2>/dev/null | sed -n 's/^[^:]*: \([^:]*\): .*/\1/p'
    }

    # An output still showing a flat colour is one nothing has drawn on. That is
    # what a fresh wl_output looks like under --no-cache.
    blank_outputs() {
      awww query 2>/dev/null |
        sed -n 's/^[^:]*: \([^:]*\): .*currently displaying: color:.*/\1/p'
    }

    draw() {
      name="$1"
      resolve_current
      # awww binds the new wl_output a beat after Hyprland announces it, and an
      # `img` naming an output it has not bound yet is dropped without error.
      for _ in $(seq 1 50); do
        if output_names | grep -qxF "$name"; then
          break
        fi
        sleep 0.1
      done
      # Same two steps as awww-init, scoped to the new output: -o keeps the
      # screens that are already correct from re-animating on every hotplug.
      awww clear 1e1e2e -o "$name"
      awww img "$current" -o "$name" ${transitionArgs}
    }

    while :; do
      sock="$XDG_RUNTIME_DIR/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket2.sock"
      if [ -S "$sock" ]; then
        # Catch-up pass before listening: a monitor attached in the gap between
        # awww-init's draw and this connect emitted a monitoradded nobody was
        # there to hear, and no later event will mention it again. Runs on every
        # reconnect too, so a Hyprland restart re-reconciles.
        for name in $(blank_outputs); do
          draw "$name"
        done

        socat -u UNIX-CONNECT:"$sock" - | while IFS= read -r line; do
          case "$line" in
            # v1 only. Hyprland emits both monitoradded>> and monitoraddedv2>>
            # for one hotplug; the v2 line does not match this pattern, which is
            # what keeps a single plug from drawing twice.
            "monitoradded>>"*) draw "''${line#monitoradded>>}" ;;
          esac
        done
      fi
      sleep 1
    done
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
        # untransitioned frame. awww-monitor-watch below covers the hotplug case
        # this gives up.
        ExecStart = "${pkgs.awww}/bin/awww-daemon --no-cache";
        ExecStartPost = awwwInit;
        MemoryAccounting = true;
        Restart = "on-failure";
        RestartSec = "1s";
      };

      Install.WantedBy = [ "hyprland-session.target" ];
    };

    systemd.user.services.awww-monitor-watch = {
      Unit = {
        Description = "Draw the current wallpaper onto hotplugged monitors";
        # After the daemon, whose ExecStartPost is awww-init: systemd holds the
        # daemon "starting" until that finishes, so the catch-up pass cannot
        # race the login reveal. PartOf drags this down with the daemon, so a
        # daemon restart re-runs the catch-up against a blank-slate compositor.
        Requires = [ "awww-daemon.service" ];
        After = [ "awww-daemon.service" ];
        PartOf = [ "awww-daemon.service" ];
      };

      Service = {
        ExecStart = awwwMonitorWatch;
        Restart = "on-failure";
        RestartSec = "1s";
      };

      # Pulled in by the daemon, not by the target. Wanting this from
      # hyprland-session.target makes the target order itself After= it, which
      # closes a cycle -- target after watch after daemon after target -- and
      # systemd breaks it by deleting the watch's start job, so it never ran.
      # A service's Wants= carries no implicit ordering, so the daemon can pull
      # this in while After= above still sequences it behind the login reveal.
      Install.WantedBy = [ "awww-daemon.service" ];
    };
  };
}
