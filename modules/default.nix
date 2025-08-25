{
  config,
  username,
  lib,
  platformConfig ? null,
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
  ] ++ lib.optionals (platformConfig != null && !platformConfig.isWSL) [
    # Only import GUI modules for non-WSL systems when platformConfig is available
  ];
  
  # Platform-aware configuration
  home.gui-settings.apps.enable = lib.mkIf (platformConfig != null) (!platformConfig.isWSL);
  
  # Don't configure home-manager here when platformConfig is used
  # This will be handled by the flake.nix home-manager configuration
  home-manager = lib.mkIf (platformConfig == null) {
    users.${username} = {
      imports = [
        ./home.nix
      ];
    };
  };
}
