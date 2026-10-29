{
  lib,
  config,
  pkgs,
  ...
}:
let
  cfg = config.programs.eww-bar;

  bar = pkgs.callPackage ./package.nix {
    inherit (cfg) colors modules defaultHidden;
    enableLaptopControls = cfg.laptopControls.enable;
    enableBattery = cfg.battery.enable;
    enableWifi = cfg.wifi.enable;
  };
  ewwBar = bar.backend;
  ewwYuck = bar.yuck;
  inherit (bar) paletteScss;

  # Drop once elkowar/eww reads StatusNotifierItem properties uncached.
  ewwPackage = pkgs.eww.overrideAttrs (oldAttrs: {
    patches = (oldAttrs.patches or [ ]) ++ [ ./eww-systray-uncached.patch ];
  });

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
      ewwPackage
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
    ])
    ++ lib.optional (cfg.ai.ccusage.package != null) cfg.ai.ccusage.package
    # Codex/Antigravity quota collection (see internal/collect/aifetch.go);
    # Claude stays on the native read-only collector.
    ++ lib.optional (cfg.ai.openusage.package != null) cfg.ai.openusage.package
    ++ [
      ewwBar
    ];

  runtimePath = lib.makeBinPath runtimePackages;

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
  options.programs.eww-bar = {
    enable = lib.mkEnableOption "the eww bar for Hyprland";

    colors = lib.mkOption {
      type = lib.types.attrsOf lib.types.str;
      default = import ./palette.nix;
      defaultText = lib.literalMD "Catppuccin Mocha, from `palette.nix`";
      description = "Colours the stylesheet reads, written to _palette.scss as one SCSS variable each";
    };

    laptopControls.enable = lib.mkEnableOption "laptop-only controls for clamshell and headless display modes";

    battery.enable = lib.mkEnableOption "the battery module and battery state collection";

    wifi.enable = lib.mkEnableOption "the Wi-Fi toggle in the network popup";

    modules = lib.mkOption {
      type = lib.types.listOf (lib.types.enum (import ./bar-modules.nix));
      default = import ./bar-modules.nix;
      defaultText = lib.literalMD "every module, from `bar-modules.nix`";
      description = "Bar modules this bar has at all. A module left out has no widget, no row in Settings > Bar modules and no collectors; the bar keeps its own order whatever the order here";
    };

    defaultHidden = lib.mkOption {
      type = lib.types.listOf (lib.types.enum (import ./bar-modules.nix));
      default = [ ];
      description = "Bar modules hidden until shown from Settings > Bar modules";
    };

    ai = {
      ccusage.package = lib.mkOption {
        type = lib.types.nullOr lib.types.package;
        default = pkgs.ccusage;
        defaultText = lib.literalExpression "pkgs.ccusage";
        description = "ccusage, for the AI usage module's token and cost totals; null leaves them out";
      };

      openusage.package = lib.mkOption {
        type = lib.types.nullOr lib.types.package;
        default = null;
        description = "openusage-cli, for the Codex and Antigravity quota cards; null leaves them out";
      };

      codexHome = lib.mkOption {
        type = lib.types.str;
        default = "${config.home.homeDirectory}/.codex";
        defaultText = lib.literalExpression ''"''${config.home.homeDirectory}/.codex"'';
        description = "CODEX_HOME for the openusage-cli probes";
      };

      claude.usageSnapshot = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        default = null;
        description = "claude-hud's display.externalUsageWritePath, read for the Claude quota card before asking the usage API; null asks the API only";
      };
    };

    blurNamespaces = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      readOnly = true;
      default = [
        "eww-bar"
        "eww-volume"
        "eww-display-mode"
        "eww-bluetooth"
        "eww-network"
        "eww-battery"
        "eww-ai-usage"
        "eww-notifications"
        "eww-wallpaper"
        "eww-settings"
        "eww-calendar"
        "eww-bar-modules"
        "eww-storage"
      ];
      description = "Layer-shell namespaces of the bar's glass surfaces, for the compositor's blur rules. The popup backdrop, a full-screen transparent click catcher, is not one of them.";
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = with pkgs; [
      baobab
      bleachbit
      ewwPackage
      ewwBar
      htop
      libappindicator
      libnotify
      pulsemixer
      socat
    ];

    xdg.configFile."eww/eww.yuck".source = ewwYuck;
    xdg.configFile."eww/eww.scss".source = ./config/eww.scss;
    xdg.configFile."eww/_palette.scss".source = paletteScss;
    xdg.configFile."eww/assets" = {
      source = ./config/assets;
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
            "EWW_BAR_MODULES=${lib.concatStringsSep "," cfg.modules}"
            "EWW_BAR_DEFAULT_HIDDEN=${lib.concatStringsSep "," cfg.defaultHidden}"
            "CODEX_HOME=${cfg.ai.codexHome}"
          ]
          ++ lib.optional (
            cfg.ai.claude.usageSnapshot != null
          ) "EWW_BAR_CLAUDE_USAGE_SNAPSHOT=${cfg.ai.claude.usageSnapshot}";
          ExecStart = "${ewwPackage}/bin/eww --force-wayland daemon --no-daemonize";
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
