{
  lib,
  config,
  pkgs,
  ...
}:
{
  imports = [
    ../options.nix
    ../../options.nix
    ./programlangs
    ./pdf
    ./secrets
    ./fonts
    ./utils
  ];
<<<<<<<< Updated upstream:home-manager/general/default.nix
  options = {
    general-settings.enable = lib.mkOption {
      type = lib.types.bool;
      default = lib.mkDefault true;
      description = "Enable my general settings";
    };
  };
  config = lib.mkIf config.general-settings.enable {
========
  config = lib.mkIf config.home.general-settings.enable {
>>>>>>>> Stashed changes:general/home/default.nix
    # All the submodules are enabled by default
    # Additional program languages I use
    home.general-settings.programlangs.packages = with pkgs; [
      gcc # c/c++
      (python3.withPackages (
        ps: with ps; [
          numpy
        ]
      )) # python
      uv # python package
      nodejs # nodejs
      jre8 # Java Development Kit
      rustc # Rust
      cargo # Rust projects
    ];
  };
}
