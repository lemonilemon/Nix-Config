{
  pkgs,
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.nixos.general.enable {
    programs.nix-ld = {
      enable = true;
      libraries = with pkgs; [
        SDL2
        SDL2_image
        SDL2_mixer
        SDL2_ttf
        wayland
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
        libX11
        libXext
        libdecor
      ];
    };
    programs.command-not-found.enable = false;
    programs.nix-index = {
      enable = true;
    };
  };
}
