{
  lib,
  config,
  pkgs,
  ...
}:
let
  cfg = config.home.desktop.hyprland.eww;

  ewwBarTools = pkgs.stdenvNoCC.mkDerivation {
    pname = "eww-bar-tools";
    version = "0";

    dontUnpack = true;
    nativeBuildInputs = [ pkgs.python3 ];

    installPhase = ''
      install -Dm755 ${./scripts/backend} $out/bin/eww-bar-backend
      cp -R ${./scripts/eww_bar_backend} $out/bin/eww_bar_backend
      cp $out/bin/eww-bar-backend $out/bin/eww-barctl
      patchShebangs $out/bin
    '';
  };

  runtimePackages =
    (with pkgs; [
      bash
      bluez
      coreutils
      eww
      gawk
      gnugrep
      gnused
      gsimplecal
      htop
      hyprland
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
    ])
    ++ [ ewwBarTools ];

  runtimePath = lib.makeBinPath runtimePackages;

  openBar = pkgs.writeShellScript "eww-open-bar" ''
    export PATH=${runtimePath}

    for _ in $(seq 1 20); do
      if eww active-windows >/dev/null 2>&1; then
        break
      fi
      sleep 0.1
    done

    eww close-all >/dev/null 2>&1 || true

    monitors=$(hyprctl monitors -j | jq -r '.[].name')
    if [ -z "$monitors" ]; then
      eww open bar
      exit 0
    fi

    for monitor in $monitors; do
      eww open bar --id "bar-$monitor" --screen "$monitor"
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

    xdg.configFile."eww/eww.yuck".source = ./eww.yuck;
    xdg.configFile."eww/eww.scss".source = ./eww.scss;
    xdg.configFile."eww/scripts".source = ./scripts;
    xdg.configFile."eww/scripts".recursive = true;

    systemd.user.services.eww-bar = {
      Unit = {
        Description = "Eww Hyprland bar";
        After = [ "hyprland-session.target" ];
        PartOf = [ "hyprland-session.target" ];
      };

      Service = {
        Environment = [ "PATH=${runtimePath}" ];
        ExecStart = "${pkgs.eww}/bin/eww --force-wayland daemon --no-daemonize";
        ExecStartPost = openBar;
        ExecStopPost = "${pkgs.eww}/bin/eww close-all";
        MemoryAccounting = true;
        Restart = "on-failure";
        RestartSec = "1s";
      };

      Install.WantedBy = [ "hyprland-session.target" ];
    };
  };
}
