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

  ewwBarTools = pkgs.stdenvNoCC.mkDerivation {
    pname = "eww-bar-tools";
    version = "0";

    dontUnpack = true;
    nativeBuildInputs = [ pkgs.python3 ];

    installPhase = ''
      install -Dm755 ${./scripts/backend} $out/bin/eww-bar-backend
      cp -R ${./scripts/eww_bar_backend} $out/bin/eww_bar_backend
      cp $out/bin/eww-bar-backend $out/bin/eww-barctl
      cp $out/bin/eww-bar-backend $out/bin/eww-popup
      patchShebangs $out/bin
    '';
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
      gsimplecal
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
      python3
      socat
      systemd
      wireplumber
      wlogout
      llm-agents.ccusage
      # Codex/Antigravity quota collection (see collectors.py); Claude stays on
      # the native read-only collector.
      openusage-cli
    ])
    ++ [ ewwBarTools ];

  runtimePath = lib.makeBinPath runtimePackages;

  ewwYuck = pkgs.replaceVars ./eww.yuck {
    laptopControls = if cfg.laptopControls.enable then "true" else "false";
    batteryModule = if cfg.battery.enable then "true" else "false";
    wifiControls = if cfg.wifi.enable then "true" else "false";
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
    # (watchers.py reacts to socket2 monitoradded/monitorremoved with the
    # same open/close commands).
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
      ewwBarTools
      gsimplecal
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
    xdg.configFile."eww/scripts".source = ./scripts;
    xdg.configFile."eww/scripts".recursive = true;

    systemd.user.services = {
      eww-bar = {
        Unit = {
          Description = "Eww Hyprland bar";
          After = [ "hyprland-session.target" ];
          PartOf = [ "hyprland-session.target" ];
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
