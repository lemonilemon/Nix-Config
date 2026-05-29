# Flake Inputs

Key inputs declared in `flake.nix`:

| Input | Purpose |
|-------|---------|
| `nixpkgs` | nixos-unstable |
| `home-manager` | follows nixpkgs |
| `nixvim` | declarative Neovim |
| `hyprland` | Wayland compositor |
| `catppuccin` | theming across NixOS/HM |
| `nixos-wsl` | WSL integration |
| `nixos-hardware` | hardware-specific modules |
| `zen-browser` | Zen browser |
| `claude-desktop` | Claude desktop for Linux |
| `grub2-themes` | bootloader themes |
| `nix-index-database` | comma / nix-index |
| `rose-pine-hyprcursor` | Hyprland cursor theme |
| `antigravity-nix` | Antigravity GUI app |
| `llm-agents` | AI coding agents (claude-code, gemini-cli, opencode, ...); daily updates, served from cache.numtide.com — overlay wired in `nixpkgs/overlays.nix`, packages live under `pkgs.llm-agents.*` |
| `template-nvim` | custom Neovim plugin (non-flake, pinned commit) |
| `coderunner-nvim` | custom Neovim plugin (non-flake, pinned commit) |
| `copilot-lualine-nvim` | custom Neovim plugin (non-flake, pinned commit) |

---

## Adding a New Input

Standard flake input:
```nix
my-input = {
  url = "github:owner/repo";
  inputs.nixpkgs.follows = "nixpkgs";  # only if the input accepts it
};
```

Non-flake source (e.g. pinned Neovim plugin):
```nix
my-plugin = {
  url = "github:owner/repo/COMMIT_HASH";
  flake = false;
};
```

All inputs become available in modules as `inputs.my-input` via `specialArgs`
(wired in by `mkSystem` / `mkHome` in `lib/builders.nix`).
