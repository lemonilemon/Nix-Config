#!/bin/sh
# Runs ON THE REMOTE, applied to the payload staged next to it by
# portable/bootstrap.sh. Needs neither Nix nor root: dotfiles go into $HOME,
# starship into ~/.local/bin, zsh plugins into ~/.local/share/zsh/plugins.
# Idempotent; a pre-existing file this script did not write is moved aside
# once as *.pre-portable.
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
plugin_dir="$HOME/.local/share/zsh/plugins"

# Every file this script writes carries "nixos-config" in its first lines;
# anything else at a destination is a real local file and gets moved aside.
place() {
  src=$1
  dest=$2
  if [ -L "$dest" ]; then
    rm -f "$dest"
  elif [ -e "$dest" ] && ! head -n 2 "$dest" 2>/dev/null | grep -q nixos-config; then
    mv "$dest" "$dest.pre-portable"
    echo "moved aside: $dest -> $dest.pre-portable"
  fi
  mkdir -p "$(dirname "$dest")"
  cp "$src" "$dest"
  echo "installed: $dest"
}

# The same agent-instruction links Home Manager maintains locally, all
# resolving to the single real file at ~/AGENTS.md.
link_agents() {
  dest=$1
  if [ -e "$dest" ] && [ ! -L "$dest" ]; then
    mv "$dest" "$dest.pre-portable"
    echo "moved aside: $dest -> $dest.pre-portable"
  fi
  mkdir -p "$(dirname "$dest")"
  ln -sf "$HOME/AGENTS.md" "$dest"
  echo "linked: $dest"
}

place "$here/zshrc" "$HOME/.zshrc"
place "$here/starship.toml" "$HOME/.config/starship.toml"
place "$here/AGENTS.md" "$HOME/AGENTS.md"

link_agents "$HOME/.claude/CLAUDE.md"
link_agents "$HOME/.codex/AGENTS.md"
link_agents "$HOME/.pi/agent/AGENTS.md"
link_agents "$HOME/.config/opencode/AGENTS.md"
# Inferred path, mirroring the Nix module: only bother if kiro-cli has run.
if [ -d "$HOME/.kiro" ]; then
  link_agents "$HOME/.kiro/steering/00-global.md"
fi

if command -v starship >/dev/null 2>&1 || [ -x "$HOME/.local/bin/starship" ]; then
  echo "starship: already present"
else
  mkdir -p "$HOME/.local/bin"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL https://starship.rs/install.sh | sh -s -- --yes --bin-dir "$HOME/.local/bin"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- https://starship.rs/install.sh | sh -s -- --yes --bin-dir "$HOME/.local/bin"
  else
    echo "warn: neither curl nor wget found; install starship by hand into ~/.local/bin" >&2
  fi
fi

# nix-zsh-completions and zsh-nix-shell are deliberately absent: no Nix here.
if command -v git >/dev/null 2>&1; then
  mkdir -p "$plugin_dir"
  clone_or_update() {
    repo=$1
    name=$2
    branch=${3:-}
    if [ -d "$plugin_dir/$name/.git" ]; then
      git -C "$plugin_dir/$name" pull --ff-only -q || echo "warn: could not update $name" >&2
    else
      # shellcheck disable=SC2086 — $branch expands to zero or two words
      git clone -q --depth 1 ${branch:+--branch "$branch"} "https://github.com/$repo" "$plugin_dir/$name"
    fi
    echo "plugin: $name"
  }
  clone_or_update marlonrichert/zsh-autocomplete zsh-autocomplete main
  clone_or_update loiccoyle/zsh-github-copilot zsh-github-copilot
else
  echo "warn: git not found; zsh plugins skipped (the zshrc degrades gracefully)" >&2
fi

zsh_bin=$(command -v zsh || true)
if [ -z "$zsh_bin" ]; then
  echo "warn: zsh not found on this machine; ~/.zshrc is installed but unused" >&2
else
  login_shell=$(getent passwd "$(id -un)" 2>/dev/null | cut -d: -f7 || true)
  [ -n "$login_shell" ] || login_shell=${SHELL:-}
  case "$login_shell" in
    */zsh | zsh) ;;
    *)
      # chsh prompts for a password, so it needs the tty bootstrap.sh allocates.
      if [ -t 0 ] && chsh -s "$zsh_bin"; then
        echo "login shell changed to $zsh_bin (takes effect on next login)"
      else
        echo "note: login shell unchanged; run: chsh -s $zsh_bin" >&2
      fi
      ;;
  esac
fi

echo "done. start a new zsh (or re-login) to pick everything up."
