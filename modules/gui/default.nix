{
  username,
  ...
}:
{
  imports = [
    ./options.nix
    ./nixos
  ];
  home-manager.users.${username} = {
    imports = [ ./home ];
  };
}
