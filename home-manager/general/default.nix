{
  lib,
  config,
  pkgs,
  ...
}:
{
  imports = [
    ./programlangs
    ./pdf
    ./secrets
    ./fonts
    ./utils
  ];
  options = {
    general-settings.enable = lib.mkOption {
      type = lib.types.bool;
      default = lib.mkDefault true;
      description = "Enable my general settings";
    };
  };
  config = lib.mkIf config.general-settings.enable {
    # All the submodules are enabled by default
    # Additional program languages I use
    general-settings.programlangs.packages = with pkgs; [
      gcc # c/c++
      python3 # python
      nodejs # nodejs
      jre8 # Java Development Kit
      rustc # Rust
      cargo # Rust projects
    ];
  };
}
