{
  pkgs,
  ...
}:
{
  imports = [
    ./nixld.nix
    ./settings.nix
    ./programlangs.nix
  ];
}
