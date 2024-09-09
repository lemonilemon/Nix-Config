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

# Run eval tests
[group('nix')]
test:
    nix eval .#evalTests --show-trace --print-build-logs --verbose

# Update all the flake inputs
[group('nix')]
update:
    nix flake update

# Update specific input

# Usage: just upp nixpkgs
[group('nix')]
upp input:
    nix flake update {{ input }}

# List all generations of the system profile
[group('nix')]
history:
    nix profile history --profile /nix/var/nix/profiles/system

# Open a nix shell with the flake
[group('nix')]
repl:
    nix repl -f flake:nixpkgs

[group('nix')]
fmt:
    # format the nix files in this repo
    nix fmt
