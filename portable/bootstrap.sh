#!/usr/bin/env bash
# Runs ON THIS MACHINE: ships the starship + zsh + agent-instruction settings
# to a remote that has neither Nix nor root. Everything lands under $HOME on
# the remote; remote-install.sh documents exactly what is written where.
#
# Usage: portable/bootstrap.sh [user@]host [extra ssh options...]
#        just portable user@host
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "usage: $0 [user@]host [extra ssh options...]" >&2
  exit 1
fi
host=$1
shift

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(dirname -- "$script_dir")

# starship.toml is taken from the rendered Home Manager file rather than the
# .nix source, so the payload always matches the running generation.
starship_toml="$HOME/.config/starship.toml"
if [ ! -r "$starship_toml" ]; then
  echo "error: $starship_toml not found; build the Home Manager generation first" >&2
  exit 1
fi

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

{
  # remote-install.sh recognizes its own files by this marker line.
  echo "# Deployed from ~/nixos-config by portable/bootstrap.sh; edit there, not here."
  cat "$starship_toml"
} >"$stage/starship.toml"
cp "$repo_root/modules/cli/home/programs/ai/AGENTS.md" "$stage/AGENTS.md"
cp "$script_dir/zshrc" "$stage/zshrc"
cp "$script_dir/remote-install.sh" "$stage/remote-install.sh"

# Two connections on purpose: the first pipes the payload as stdin, and the
# second allocates a tty so chsh can prompt for a password.
remote_dir='.cache/nixos-config-portable'
tar -C "$stage" -czf - . |
  ssh "$@" "$host" "rm -rf '$remote_dir' && mkdir -p '$remote_dir' && tar -xzf - -C '$remote_dir'"
ssh -t "$@" "$host" "sh '$remote_dir/remote-install.sh'"
