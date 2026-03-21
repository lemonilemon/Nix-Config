# Modules

This directory contains all modular configuration components. Each module controls a specific concern and is gated by a feature flag.

## Structure

```
modules/
├── options.nix   # All top-level feature flag declarations
├── home.nix      # Home Manager entry point
├── nixos.nix     # NixOS base configuration
├── default.nix   # Aggregates all modules
├── cli/          # CLI tools: Neovim, shells, git, Zellij, AI tools
├── desktop/      # Desktop environments: Hyprland, GNOME, SDDM
├── general/      # System settings: fonts, languages, utilities
└── gui/          # GUI apps: browsers, terminal emulator, applications
```

## Feature Flags

All flags are declared in `options.nix` and default to `true`:

| Option | Description |
|--------|-------------|
| `home.enable` | Home Manager configuration |
| `nixos.enable` | NixOS system configuration |
| `cli.enable` | CLI tools and terminal configuration |
| `gui.enable` | GUI applications |
| `general.enable` | General system settings |
| `desktop.enable` | Desktop environment |

Sub-options follow the same pattern (e.g. `home.cli.nvim.enable`, `home.gui.browsers.firefox.enable`).

## Adding a New Module

1. Create `modules/<category>/<feature>.nix` using the standard skeleton:
   ```nix
   { config, lib, pkgs, ... }:
   let cfg = config.<option>; in
   {
     config = lib.mkIf cfg.enable {
       # configuration here
     };
   }
   ```
2. Declare any new feature flags in `options.nix`.
3. Import the new file in the category's `default.nix`.

See each subdirectory's README for detailed documentation.
