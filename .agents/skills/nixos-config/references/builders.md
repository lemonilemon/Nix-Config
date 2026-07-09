# Builders (`lib/builders.nix`)

The library exposes three functions used to build configurations.

---

## `mkSystem` — NixOS System Builder

Builds a complete NixOS system, integrating Home Manager, all modules, overlays, and a profile.

### Signature

```nix
mkSystem {
  system          # e.g. "x86_64-linux"
  username
  hostname
  profile         # directory name under profiles/
  isWSL    ? false   # adds nixos-wsl module; skips grub2-themes
  isDarwin ? false
  extraModules ? []  # additional NixOS modules to splice in
}
```

### What it wires in automatically

- `modules/` — all custom NixOS + Home Manager modules
- `overlays/` — repo overlays (see `overlays/` and `nixpkgs/overlays.nix` for the current list)
- `catppuccin.nixosModules.catppuccin`
- Home Manager with `useGlobalPkgs = true`, `useUserPackages = false`
  - HM user modules: `modules/home.nix`, catppuccin HM, nix-index-database, nixvim
- `profiles/<profile>/` — machine-specific config
- `grub2-themes` (standalone only, skipped for WSL/Darwin)
- `nixos-wsl` (WSL only)

### `specialArgs` passed to all modules

`inputs`, `username`, `hostname`, plus all other `mkSystem` args.

---

## `mkHome` — Standalone Home Manager Builder

Intended for non-NixOS systems where only Home Manager is needed.

> **Currently unused and non-functional**: nothing in `flake.nix` calls `mkHome`, and it
> imports `../home-manager`, a directory that does not exist yet. Do not suggest it as a
> working path; if the user wants standalone Home Manager, the `home-manager/` entry point
> must be created first.

### Signature

```nix
mkHome {
  system
  username
  pkgs
  isWSL        ? false
  extraModules ? []
}
```

### What it wires in

- `../home-manager` (standalone path, separate from `modules/home.nix`)
- nixvim, catppuccin HM, nix-index-database
- `extraSpecialArgs`: `inputs`, `username`, `system`, `helpers.mkHomeOpt`

---

## `mkHomeOpt` — Option Mirroring Helper

Used in Home Manager `options.nix` files to declare options that automatically inherit their value
from the parent NixOS system's `osConfig` when available, falling back to `default` when running
standalone (no NixOS parent).

### Signature

```nix
helpers.mkHomeOpt {
  osConfig     # pass-through from HM module args (null in standalone HM)
  path         # dot-string path into osConfig, e.g. "home.cli.nvim.enable"
  default      # fallback when osConfig is null or path not found
  description
  type    ? lib.types.bool
}
```

### Example

```nix
{ config, osConfig ? null, helpers, ... }:
{
  options.home.cli.nvim.enable = helpers.mkHomeOpt {
    inherit osConfig;
    path = "home.cli.nvim.enable";
    default = config.home.cli.enable;
    description = "Enable nvim settings";
  };
}
```

This pattern means that when running inside NixOS, Home Manager options automatically mirror
whatever the NixOS config has set — no need to set them twice.
