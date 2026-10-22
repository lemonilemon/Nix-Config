{
  lib,
  config,
  pkgs,
  inputs,
  ...
}:
let
  waylandSessions = pkgs.runCommandLocal "sddm-wayland-sessions" { } ''
    mkdir $out
    ${pkgs.lndir}/bin/lndir -silent ${config.services.displayManager.sessionData.desktops}/share/wayland-sessions $out
    rm -f $out/hyprland-uwsm.desktop
  '';

  orrery = inputs.sddm-orrery.packages.${pkgs.stdenv.hostPlatform.system}.default.override {
    osLabel = "NixOS ${config.system.nixos.release} ${config.system.nixos.codeName}";
  };
in
{
  config = lib.mkIf (config.nixos.desktop.enable && config.nixos.desktop.displayManager == "sddm") (
    lib.mkMerge [
      {
        services.displayManager.sddm = {
          enable = true;
          wayland = {
            enable = true;
            compositor = "kwin";
          };
          autoNumlock = true;
          package = pkgs.kdePackages.sddm;
          enableHidpi = true;
          settings = {
            Theme = {
              CursorTheme = "BreezeX-RosePine-Linux";
              CursorSize = 24;
            };
            Wayland.SessionDir = "${waylandSessions}";
          };
        };
        environment.systemPackages = [ pkgs.rose-pine-cursor ];
        catppuccin.sddm.enable = lib.mkDefault false;
      }

      (lib.mkIf (config.nixos.desktop.sddmTheme == "orrery") {
        services.displayManager.sddm.theme = "orrery";
        environment.systemPackages = [ orrery ];
        fonts.packages = [ orrery.fonts ];
      })

      (lib.mkIf (config.nixos.desktop.sddmTheme == "astronaut") {
        services.displayManager.sddm = {
          theme = "sddm-astronaut-theme";
          extraPackages = with pkgs; [
            kdePackages.qtsvg
            kdePackages.qtvirtualkeyboard
            kdePackages.qtmultimedia
          ];
        };
        environment.systemPackages = [ pkgs.sddm-astronaut ];
      })
    ]
  );
}
