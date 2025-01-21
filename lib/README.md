# Library
## Introduction
- `attrs.nix`: A set of functions to manipulate attribute sets.
- `hmSystem.nix`: A function to generate config (attribute set) for non-NixOS Linux distribution which uses home-manager to manage their dotfiles.
- `nixosSystem.nix`: A function to generate config (attribute set) for NixOS.
- `default.nix`: import all the above functions, and export them as a single attribute set.

## Attribution
Most of these library functions are from [ryan4yin/nix-config](https://github.com/ryan4yin/nix-config/blob/main/lib) to get rid of redundant code.
