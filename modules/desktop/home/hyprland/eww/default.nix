{
  lib,
  config,
  pkgs,
  ...
}:
let
  cfg = config.home.desktop.hyprland.eww;

  palette = import ../theme/palette.nix;
  paletteScss = pkgs.writeText "_palette.scss" (
    lib.concatStringsSep "\n" (
      lib.mapAttrsToList (name: value: "\$" + name + ": " + value + ";") palette
    )
  );

  # The daemon, eww-barctl and eww-popup, one Go module.
  #
  # The daemon was Python until the port: 93% of its CPU is the child processes
  # its collectors fork (pactl, nmcli, hyprctl, ...), which cost the same in any
  # language, so the interpreter was never the expense. What moved it was the
  # CLIENTS -- eww.yuck invokes them from 39 handlers including :onscroll, and
  # they do almost no work: the daemon answers a ping in 0.06 ms against ~45 ms
  # of CPython startup spent asking it. Measured: 45.4 ms -> 3.3 ms for
  # eww-barctl, 50.8 ms -> 3.3 ms for eww-popup. Once those were compiled,
  # leaving the daemon behind meant maintaining the same collectors twice.
  ewwBar = pkgs.buildGoModule {
    pname = "eww-bar";
    version = "0";

    src = ./go;

    # Every import is stdlib, so there is nothing to vendor. vendorHash = null
    # creates no fetch derivation at all, which is the point: no hash to keep in
    # step with a dependency set that does not exist.
    vendorHash = null;

    # No subPackages: it would narrow checkPhase to cmd/, which holds no tests,
    # and silently skip every internal package. Building ./... installs the one
    # main package under cmd/ and runs `go test ./...` over the rest.

    postInstall = ''
      # argv[0] dispatch, as the Python entry point used to do: exec'ing a
      # symlink leaves argv[0] as the name the caller typed.
      ln -s eww-bar-client $out/bin/eww-barctl
      ln -s eww-bar-client $out/bin/eww-popup
    '';

    meta.mainProgram = "eww-bar-backend";
  };

  runtimePackages =
    (with pkgs; [
      awww
      bash
      bluez
      blueman
      # The network popup's speed test (see internal/collect/speedtest.go).
      # cfspeedtest rather than speedtest-cli or speedtest-go: both of those
      # fetch speedtest.net's config endpoint, which now answers an
      # unauthenticated GET with a Cloudflare bot challenge, so they fail before
      # transferring a byte. Ookla's own client works and is unfree, needs a
      # licence acknowledgement on every run and has no binary cache.
      cfspeedtest
      coreutils
      dbus
      dunst
      eww
      gawk
      gnugrep
      gnused
      htop
      hyprland
      imagemagick
      iproute2
      jq
      kitty
      networkmanager
      playerctl
      pipewire
      procps
      pulseaudio
      pulsemixer
      socat
      systemd
      wireplumber
      wlogout
      llm-agents.ccusage
      # Codex/Antigravity quota collection (see internal/collect/aifetch.go);
      # Claude stays on the native read-only collector.
      openusage-cli
    ])
    ++ [
      ewwBar
    ];

  runtimePath = lib.makeBinPath runtimePackages;

  ewwYuck = pkgs.replaceVars ./eww.yuck {
    laptopControls = if cfg.laptopControls.enable then "true" else "false";
    batteryModule = if cfg.battery.enable then "true" else "false";
    wifiControls = if cfg.wifi.enable then "true" else "false";
    # defwindow takes a fixed size, and network_panel has nothing that expands,
    # so a window sized for the Wi-Fi row would leave a dead band inside the
    # card once that row is hidden. The row is ~40px including its margin.
    #
    # Still only two values after the speed card, which is the whole reason that
    # card renders in every state instead of appearing after a run: a
    # conditional card would have crossed with this and needed four.
    #
    # +114px over the original 200/160, measured rather than derived: the same
    # popup was rendered against a scratch eww daemon with and without the new
    # content, in a deliberately oversized window, and the footer's baseline
    # compared. GTK's natural heights do not agree with the arithmetic -- adding
    # up the min-heights and paddings under-predicted and would have clipped the
    # footer.
    #
    # 86 of it is the speed card and 28 the restrictions line. That line was a
    # row of SSH/DNS/IPv6 chips first, which cost 69px; folding it into the speed
    # card as one sentence gave 41px back and removed the legend nobody had.
    networkPopupHeight = if cfg.wifi.enable then "314px" else "274px";
  };

  openBar = pkgs.writeShellScript "eww-open-bar" ''
    export PATH=${runtimePath}

    daemon_ready=false
    for _ in $(seq 1 100); do
      if eww active-windows >/dev/null 2>&1; then
        daemon_ready=true
        break
      fi
      sleep 0.1
    done

    # bail rather than open against a daemon that never came up
    if [ "$daemon_ready" != true ]; then
      echo "eww daemon not reachable" >&2
      exit 1
    fi

    eww close-all >/dev/null 2>&1 || true

    # --no-daemonize belongs on every `eww open` this config issues -- here, in
    # collect.BarWindowCommand and in internal/popup -- and it is the only thing
    # standing between a slow daemon and a SECOND bar.
    #
    # eww's client starts a daemon of its own whenever a server action fails,
    # gated on `action.can_start_daemon() && !opts.no_daemonize`, and `open` is
    # one of the two actions that can. The failure that fires it is not "no
    # daemon is running": the client gives the daemon ~100 ms to answer, and
    # building this bar's widget tree takes longer than that on a loaded
    # machine. The daemon opens the window regardless, so what the client reads
    # as a failure is really a slow success -- and the fallback then rebinds the
    # IPC socket, reloads the config, and spawns an eww-bar-backend of its own,
    # whose ReconcileBarWindows opens a bar into the new daemon. Two stacked
    # bars, two backends, and the systemd-managed daemon left unreachable behind
    # a socket it no longer owns. Seen on 2026-08-20 with a speed test running.
    #
    # With the flag that timeout is an ordinary non-zero exit, which the retry
    # below can react to instead of eww silently forking.
    open_bar() {
      bar_id=$1
      shift
      for _ in 1 2 3 4 5; do
        # Re-checked before every attempt, not just the first: a timed-out open
        # has usually already opened the window, and blindly re-issuing it is
        # itself a way to end up with two.
        if eww active-windows 2>/dev/null | grep -q "^$bar_id:"; then
          return 0
        fi
        if eww --no-daemonize open bar "$@"; then
          return 0
        fi
        sleep 0.5
      done
      echo "could not open bar window $bar_id" >&2
      return 1
    }

    # Startup only — monitors hotplugged later are handled by the backend
    # (internal/watch's WatchHyprland reacts to socket2
    # monitoradded/monitorremoved with the same open/close commands).
    monitors=$(hyprctl monitors -j | jq -r '.[].name')
    if [ -z "$monitors" ]; then
      # No --id, so the window lands under its own name, "bar".
      open_bar bar --arg output=0
      exit $?
    fi

    # rc, not `status`: writeShellScript is bash, where the name is free, but
    # zsh reserves it as a read-only alias for $? and this script is short
    # enough to be worth pasting into a shell while debugging.
    rc=0
    for monitor in $monitors; do
      open_bar "bar-$monitor" \
        --id "bar-$monitor" --screen "$monitor" --arg "output=$monitor" || rc=1
    done
    exit $rc
  '';
in
{
  config = lib.mkIf cfg.enable {
    home.packages = with pkgs; [
      eww
      ewwBar
      htop
      libappindicator
      libnotify
      pulsemixer
      socat
    ];

    xdg.configFile."eww/eww.yuck".source = ewwYuck;
    xdg.configFile."eww/eww.scss".source = ./eww.scss;
    xdg.configFile."eww/_palette.scss".source = paletteScss;
    xdg.configFile."eww/assets" = {
      source = ./assets;
      recursive = true;
    };

    systemd.user.services = {
      eww-bar = {
        Unit = {
          Description = "Eww Hyprland bar";
          After = [ "hyprland-session.target" ];
          PartOf = [ "hyprland-session.target" ];
          # sd-switch only restarts a service whose unit file changed. Without
          # this, a yuck-only change (anything gated purely in the template,
          # e.g. the Wi-Fi toggle) would leave the running bar on the old
          # config until the next login.
          X-Restart-Triggers = [ "${ewwYuck}" ];
        };

        Service = {
          Environment = [
            "PATH=${runtimePath}"
            "EWW_BAR_BATTERY=${if cfg.battery.enable then "1" else "0"}"
          ];
          ExecStart = "${pkgs.eww}/bin/eww --force-wayland daemon --no-daemonize";
          ExecStartPost = openBar;
          MemoryAccounting = true;
          Restart = "on-failure";
          RestartSec = "1s";
        };

        Install.WantedBy = [ "hyprland-session.target" ];
      };

      eww-hypridle-inhibit = {
        Unit.Description = "Eww Hypridle inhibitor";
        Service = {
          Type = "simple";
          Environment = [ "PATH=${runtimePath}" ];
          ExecStart = "${pkgs.systemd}/bin/systemd-inhibit --what=idle --who=eww-bar --why=\"Hypridle paused from Eww\" ${pkgs.coreutils}/bin/sleep infinity";
        };
      };
    }
    // lib.optionalAttrs cfg.laptopControls.enable {
      eww-lid-inhibit = {
        Unit.Description = "Eww laptop lid inhibitor";
        Service = {
          Type = "simple";
          Environment = [ "PATH=${runtimePath}" ];
          ExecStart = "${pkgs.systemd}/bin/systemd-inhibit --what=handle-lid-switch --who=eww-bar --why=\"Eww laptop display mode keeps lid close ignored\" ${pkgs.coreutils}/bin/sleep infinity";
        };
      };
    };
  };
}
