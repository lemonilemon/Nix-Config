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
  home.gui-settings.apps.enable = false;
  home-manager.users.${username} = {
    imports = [
      ./home.nix
    ];
  };
}
