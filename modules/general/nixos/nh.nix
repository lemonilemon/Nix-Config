{
  lib,
  pkgs,
  config,
  ...
}:
let
  cfg = config.nixos.general.nh;
  cleanArgs = "--keep ${toString cfg.clean.keep} --keep-since ${cfg.clean.keepSince}";
  cleanNow = pkgs.writeShellScriptBin "nh-clean-now" ''
    exec ${lib.getExe config.programs.nh.package} clean all --ask ${cleanArgs} "$@"
  '';
in
{
  config = lib.mkIf cfg.enable {
    programs.nh = {
      enable = true;
      inherit (cfg) flake;
      clean = {
        inherit (cfg.clean) enable;
        extraArgs = cleanArgs;
      };
    };

    environment.systemPackages = [ cleanNow ];
  };
}
