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
    networkPopupHeight = if cfg.wifi.enable then "200px" else "160px";
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

    # bail rather than let `eww open` fork a rogue daemon
    if [ "$daemon_ready" != true ]; then
      echo "eww daemon not reachable" >&2
      exit 1
    fi

    eww close-all >/dev/null 2>&1 || true

    # Startup only — monitors hotplugged later are handled by the backend
    # (internal/watch's WatchHyprland reacts to socket2
    # monitoradded/monitorremoved with the same open/close commands).
    monitors=$(hyprctl monitors -j | jq -r '.[].name')
    if [ -z "$monitors" ]; then
      eww open bar --arg output=0
      exit 0
    fi

    for monitor in $monitors; do
      eww open bar --id "bar-$monitor" --screen "$monitor" --arg "output=$monitor"
    done
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
