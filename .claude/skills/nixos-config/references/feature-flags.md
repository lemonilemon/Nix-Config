# Feature Flags

All options are declared in `modules/options.nix` (top-level) and category `options.nix` files.
The option tree mirrors the NixOS module system. Sub-options inherit from their parent by default.

## Top-level (`modules/options.nix`)

| Option | Default | Description |
|--------|---------|-------------|
| `home.enable` | `true` | Enable Home Manager configuration |
| `nixos.enable` | `true` | Enable NixOS system configuration |
| `cli.enable` | `true` | Enable CLI tools |
| `gui.enable` | `true` | Enable GUI applications |
| `general.enable` | `true` | Enable general system settings |
| `desktop.enable` | `true` | Enable desktop environment |

---

## CLI (`modules/cli/options.nix`)

| Option | Default |
|--------|---------|
| `home.cli.enable` | `home.enable && cli.enable` |
| `home.cli.git.enable` | `home.cli.enable` |
| `home.cli.nvim.enable` | `home.cli.enable` |
| `home.cli.programs.enable` | `home.cli.enable` |
| `home.cli.shells.enable` | `home.cli.enable` |
| `home.cli.shells.zsh.enable` | `home.cli.shells.enable` |
| `home.cli.shells.starship.enable` | `home.cli.shells.enable` |
| `home.cli.shells.zoxide.enable` | `home.cli.shells.enable` |
| `home.cli.multiplexer.program` | `"tmux"` (enum: `"tmux"` \| `"zellij"`) |
| `home.cli.multiplexer.zellij.enable` | `multiplexer.program == "zellij"` |
| `home.cli.multiplexer.tmux.enable` | `multiplexer.program == "tmux"` |
| `nixos.cli.enable` | `nixos.enable && cli.enable` |

---

## GUI (`modules/gui/options.nix`)

| Option | Default |
|--------|---------|
| `home.gui.enable` | `home.enable && gui.enable` |
| `home.gui.browsers.enable` | `home.gui.enable` |
| `home.gui.browsers.firefox.enable` | `home.gui.browsers.enable` |
| `home.gui.browsers.zen.enable` | `home.gui.browsers.enable` |
| `home.gui.apps.enable` | `home.gui.enable` |
| `home.gui.kitty.enable` | `home.gui.enable` |
| `home.gui.development.enable` | `home.gui.enable` |
| `home.gui.development.web.enable` | `home.gui.development.enable` |
| `nixos.gui.enable` | `nixos.enable && gui.enable` |
| `nixos.gui.apps.enable` | `nixos.gui.enable` |
| `nixos.gui.development.enable` | `nixos.gui.enable` |
| `nixos.gui.development.web.enable` | `nixos.gui.development.enable` |

---

## Desktop (`modules/desktop/options.nix`)

| Option | Default |
|--------|---------|
| `home.desktop.enable` | `home.enable && desktop.enable` |
| `home.desktop.hyprland.enable` | `home.desktop.enable` |
| `nixos.desktop.enable` | `nixos.enable && desktop.enable` |
| `nixos.desktop.hyprland.enable` | `nixos.desktop.enable` |
| `nixos.desktop.gnome.enable` | `nixos.desktop.enable` |
| `nixos.desktop.displayManager` | `"sddm"` (string) |

---

## General (`modules/general/options.nix`)

| Option | Default |
|--------|---------|
| `home.general.enable` | `home.enable && general.enable` |
| `home.general.fonts.enable` | `home.general.enable` |
| `home.general.pdf.enable` | `home.general.enable` |
| `home.general.programlangs.enable` | `home.general.enable` |
| `home.general.programlangs.packages` | `[ pkgs.gcc pkgs.python3 ]` |
| `home.general.secrets.enable` | `home.general.enable` |
| `home.general.utils.enable` | `home.general.enable` |
| `nixos.general.enable` | `nixos.enable && general.enable` |
| `nixos.general.nix.enable` | `nixos.general.enable` |
| `nixos.general.nixld.enable` | `nixos.general.enable` |

---

## `mkHomeOpt` — Option Mirroring

Home Manager modules use `helpers.mkHomeOpt` instead of `lib.mkOption` for options that should
mirror the parent NixOS system's value when running within a NixOS system. When running standalone
(no `osConfig`), the `default` value is used instead. See [`builders.md`](builders.md) for details.
