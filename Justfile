# This just file is from Ryan Yin (https://github.com/ryan4yin/nix-config/blob/main/Justfile)

set shell := ["zsh", "-c"]

############################################################################
#
#  Common commands(suitable for all machines)
#
############################################################################

# List all the just commands
default:
    @just --list

# Update all the flake inputs
[group('nix')]
update:
    nix flake update

# Update specific input

# Usage: just upp nixpkgs
[group('nix')]
updatep input:
    nix flake update {{ input }}

# List all generations of the system profile
[group('nix')]
history:
    nix profile history --profile /nix/var/nix/profiles/system

# Open a nix shell with the flake
[group('nix')]
repl:
    nix repl -f flake:nixpkgs

# format the nix files in this repo
[group('nix')]
fmt:
    nix fmt

############################################################################
#
#  NixOS commands
#
############################################################################

nixhost := x"${NIXHOST}"
username := env_var_or_default("USER", "lemonilemon")
# build system with new config
[group('NixOS')]
build:
    sudo nixos-rebuild switch --flake .#{{ nixhost }}

[group('NixOS')]
dry-build:
    sudo nixos-rebuild dry-build --flake .#{{ nixhost }}

[group('NixOS')]
changehost input:
    export NIXHOST={{ input }}
    sudo nixos-rebuild switch --flake .#{{ input }}

# Run eval tests
[group('NixOS')]
test:
    nix eval .#nixosConfigurations.{{ nixhost }}.config.system.build.toplevel.drvPath --show-trace

[group('NixOS')]
gc:
    sudo nix-collect-garbage -d

# Build current host's toplevel and push its closure to <username>.cachix.org.
# Requires `cachix authtoken <token>` to have been run once.
[group('NixOS')]
push:
    cachix watch-exec {{ username }} -- \
      nix build .#nixosConfigurations.{{ nixhost }}.config.system.build.toplevel --no-link --print-out-paths
