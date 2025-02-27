{
  pkgs,
  ...
}:
{
  config = {
    programs.nix-ld = {
      enable = true;
      libraries = with pkgs; [
        stdenv.cc.cc
        zlib
        fuse3
        icu
        nss
        openssl
        curl
        expat
        libcxx
        glibc
        musl
        glibc.static
        zlib.static
        libtool
      ];
    };
  };
}
