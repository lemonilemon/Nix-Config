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
    ./rime
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
      micromamba # python virtualenv
      nodejs # nodejs
      bun
      jre8 # Java Development Kit
      rustc # Rust
      cargo # Rust projects
    ];
    home.sessionVariables = {
      MAMBA_ROOT_PREFIX = "${config.home.homeDirectory}/.micromamba";
    };
    programs.zsh.initContent = ''
      # nixpkgs wraps the micromamba binary as `.mamba-wrapped`. The upstream
      # shell hook derives its function name from /proc/self/exe's basename and
      # strips the extension with ''${name%.*}; the leading dot makes that expand
      # to an empty string, so the `micromamba` function is never defined and it
      # prints "Error unknown MAMBA_EXE ... filename must be mamba or micromamba".
      # Suppress that error and define the function as a fallback.
      eval "$(micromamba shell hook --shell zsh)" 2>/dev/null
      if ! typeset -f micromamba > /dev/null; then
        micromamba() { __mamba_wrap "$@"; }
      fi
    '';
  };
}
