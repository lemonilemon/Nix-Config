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
        linuxPackages.nvidia_x11
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
        libxcrypt
      ];
    };
    programs.command-not-found.enable = false;
    programs.nix-index = {
      enable = true;
    };
  };
}
