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
  config = lib.mkIf config.home.general-settings.enable {
    # All the submodules are enabled by default
    # Additional program languages I use
    home.general-settings.programlangs.packages = with pkgs; [
      gcc # c/c++
      python3
      # (python3.withPackages (
      #   ps: with ps; [
      #     # numpy
      #   ]
      # )) # python
      uv # python package
      nodejs # nodejs
      jre8 # Java Development Kit
      rustc # Rust
      cargo # Rust projects
    ];
  };
}
