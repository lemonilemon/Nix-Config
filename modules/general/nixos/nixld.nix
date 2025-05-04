{
  pkgs,
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.nixos.general-settings.enable {
    programs.nix-ld = {
      enable = true;
      libraries = with pkgs; [
        cudatoolkit # For CUDA support
        stdenv.cc.cc
        fuse3
        icu
        nss
        openssl
        curl
        expat
        libcxx
        glib
        glibc
        glibc.static
        zlib
        zlib.static
        musl
        libtool
        libGL
      ];
    };
    programs.command-not-found.enable = false;
    programs.nix-index = {
      enable = true;
    };
  };
}
