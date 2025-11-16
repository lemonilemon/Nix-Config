{
  config,
  username,
  ...
}:
{
  imports = [
    ./options.nix
    ./nixos.nix
    ./cli
    ./gui
    ./general
    ./desktop
  ];
  home-manager.users.${username} = {
    imports = [
      ./home.nix
    ];
  };
}
