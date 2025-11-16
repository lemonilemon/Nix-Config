{
  lib,
  config,
  pkgs,
  ...
}:
{
  imports = [
    ./options.nix
    ./programlangs
    ./pdf
    ./secrets
    ./fonts
    ./utils
  ];
  config = lib.mkIf config.home.general.enable {
    # All the submodules are enabled by default
    # Additional program languages I use
    home.general.programlangs.packages = with pkgs; [
      gcc # c/c++
      (python3.withPackages (
        ps: with ps; [
          numpy
        ]
      )) # python
      uv # python package
      # micromamba # python virtualenv
      nodejs # nodejs
      bun
      jre8 # Java Development Kit
      rustc # Rust
      cargo # Rust projects
    ];
  };
}
