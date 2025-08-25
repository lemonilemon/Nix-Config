{ config, lib, pkgs, platformConfig, username, inputs, ... }: {
  imports = [
    # Import existing home manager modules
    ../modules/home.nix
    
    # Import catppuccin for home-manager
    inputs.catppuccin.homeModules.catppuccin
    inputs.nix-index-database.homeModules.nix-index
    
    # Platform-conditional imports
  ] ++ lib.optionals (!platformConfig.isWSL) [
    # GUI applications and desktop configuration only for non-WSL
  ];

  # Don't override home settings here since they're handled by modules/home.nix
  # Enable nix-index for command-not-found functionality  
  programs.nix-index = {
    enable = true;
    enableZshIntegration = true;
  };
}