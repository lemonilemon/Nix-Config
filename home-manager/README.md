# Home-manager

## Overview
This directory contains all the definition of Home-manager configuration, which is used in both NixOS and non-NixOS distros that uses Home-manager to manage dotfiles. I use a modular structure for better organization, all the defined options here are enabled by default.

---

## File Structure
```bash
.
├── default.nix # Options are enabled here
├── cli         # CLI-settings
├── general     # general-settings
└── gui         # GUI-settings
```
