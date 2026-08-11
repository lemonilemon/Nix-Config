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

# Ship starship + zsh + agent instructions to a remote without Nix or root
[group('remote')]
portable host *ssh_opts:
    ./portable/bootstrap.sh {{ host }} {{ ssh_opts }}

# Default empty rather than erroring on unset: bootstrap recipes (key-fetch)
# must run on a fresh machine before any rebuild has set NIXHOST.
nixhost := env_var_or_default("NIXHOST", "")
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

# Run flake checks (nixfmt, host option assertions, backend tests)
[group('nix')]
check:
    nix flake check

# Run just the eww backend tests, without the full flake check.
# PYTHONDONTWRITEBYTECODE mirrors the sandboxed check and keeps __pycache__
# dirs out of the tree; they are gitignored but still clutter it.
[group('nix')]
test-backend:
    PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests -t . -v

# Run eval tests
[group('NixOS')]
test:
    nix eval .#nixosConfigurations.{{ nixhost }}.config.system.build.toplevel.drvPath --show-trace

[group('NixOS')]
gc:
    sudo nix-collect-garbage -d

############################################################################
#
#  Secrets (sops + age; the age key is backed up in 1Password)
#
############################################################################

# The 1Password item holding the sops age key. Lookups go through the
# immutable item ID so retitling the item cannot break them; the title and
# vault are only used when key-backup creates the item for the first time.
# If you ever delete and recreate the item, update op_key_id.
op_key_id := "6oagih2m65vfcpucuc7kpqcrp4"
op_key_title := "SOPS age key (nixos-config)"
op_key_vault := "Development"
age_key_file := x"$HOME/.config/sops/age/keys.txt"

# Edit a sops-encrypted secret by name, whatever its extension: `just secrets
# ssh-hosts` opens secrets/ssh-hosts.conf. With no name, prints the inventory
# (secrets/README.md) instead. Creating a brand-new secret file is the one
# case this can't resolve — run `sops secrets/<file>` directly for that.
[group('secrets')]
[doc("Edit a sops-encrypted secret by name; with no name, list them")]
secrets name="":
    #!/usr/bin/env zsh
    set -euo pipefail
    if [[ -z "{{ name }}" ]]; then
        cat secrets/README.md
        exit 0
    fi
    matches=(secrets/{{ name }}(N) secrets/{{ name }}.*(N))
    if (( ${#matches} == 0 )); then
        echo "no file matching secrets/{{ name }}*" >&2
        exit 1
    elif (( ${#matches} > 1 )); then
        echo "ambiguous name, matches: ${matches[*]}" >&2
        exit 1
    fi
    if command -v sops > /dev/null; then
        sops "${matches[1]}"
    else
        nix shell nixpkgs#sops -c sops "${matches[1]}"
    fi

# Back up the age key to 1Password (updates the item if it already exists)
[group('secrets')]
key-backup:
    op document edit "{{ op_key_id }}" "{{ age_key_file }}" 2> /dev/null || \
      op document create "{{ age_key_file }}" --title "{{ op_key_title }}" --vault "{{ op_key_vault }}"

# Restore the age key from 1Password (refuses to overwrite an existing key)
[group('secrets')]
key-fetch:
    #!/usr/bin/env zsh
    set -euo pipefail
    key="{{ age_key_file }}"
    if [[ -e "$key" ]]; then
        echo "$key already exists; delete it first to re-fetch." >&2
        exit 1
    fi
    mkdir -p "${key:h}" && chmod 700 "${key:h}"
    umask 177
    op document get "{{ op_key_id }}" > "$key"
    echo "age key restored to $key"

# Build current host's toplevel and push its closure to <username>.cachix.org.
# Requires `cachix authtoken <token>` to have been run once.
[group('NixOS')]
push:
    cachix watch-exec {{ username }} -- \
      nix build .#nixosConfigurations.{{ nixhost }}.config.system.build.toplevel --no-link --print-out-paths
