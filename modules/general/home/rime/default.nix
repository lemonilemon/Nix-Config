{
  lib,
  config,
  ...
}:
let
  rimeDir = "${config.xdg.dataHome}/fcitx5/rime";
in
{
  config = lib.mkIf config.home.general.rime.enable {
    xdg.dataFile."fcitx5/rime/default.custom.yaml" = {
      text = ''
        patch:
          schema_list:
            - schema: bopomofo_tw
      '';
      # Rime checks file modification time (mtime) to decide whether to rebuild its cache.
      # Since Nix store symlinks have an mtime of 0 (Unix epoch), Rime assumes the config
      # hasn't changed and skips auto-deployment. Deleting the build cache forces Rime
      # to re-evaluate the config on the next startup.
      onChange = "rm -rf ${rimeDir}/build";
    };
  };
}
