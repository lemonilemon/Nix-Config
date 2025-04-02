{
  pkgs,
  ...
}:
{
  environment.systemPackages = with pkgs; [
    stdenv.cc.cc
    zlib
  ];
}
