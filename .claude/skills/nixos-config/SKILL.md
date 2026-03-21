---
name: nixos-config
description: >
  Manages a personal NixOS configuration (flakes + Home Manager + modular).
  Use this skill whenever the user asks to add, edit, or remove NixOS options, Home Manager options,
  flake inputs, overlays, or modules in their config. Also use when the user asks how to build,
  test, or rebuild their system, where to put a new package or service, or how a specific option
  is named or typed. Trigger on phrases like "add X to my nix config", "enable X in NixOS",
  "configure X with home-manager", "add a flake input", "where do I put X", "how do I set option X",
  "update my flake", or any mention of editing .nix files in their config.
---

# NixOS Config Skill

## First Step

Orient with the repo structure before acting:
- Top-level layout and profiles → [README.md](../../README.md)
- Module system and feature flags overview → [modules/README.md](../../modules/README.md)

Then, if the file tree isn't already visible:
```bash
ls && cat flake.nix
```

---

## References

| Topic | Source |
|-------|--------|
| Directory layout, profiles, quick start | [`README.md`](../../README.md) |
| Module system, adding modules | [`modules/README.md`](../../modules/README.md) |
| Full feature flag hierarchy | [`references/feature-flags.md`](references/feature-flags.md) |
| Flake inputs | [`references/flake-inputs.md`](references/flake-inputs.md) |
| `mkSystem` / `mkHome` / `mkHomeOpt` | [`references/builders.md`](references/builders.md) |
| Nix language & module syntax | [`references/nix-syntax.md`](references/nix-syntax.md) |
| CLI module (Neovim, shells, AI tools) | [`modules/cli/README.md`](../../modules/cli/README.md) |
| GUI module (browsers, apps, kitty) | [`modules/gui/README.md`](../../modules/gui/README.md) |
| Desktop module (Hyprland, GNOME) | [`modules/desktop/README.md`](../../modules/desktop/README.md) |
| General module (fonts, langs, utils) | [`modules/general/README.md`](../../modules/general/README.md) |

---

## Modular Philosophy

Options are declared centrally in `modules/options.nix`; modules implement them via `lib.mkIf`. Profiles only set option values — they do not contain inline NixOS/HM config.

**Adding options that don't exist yet**: always add to `modules/options.nix` explicitly using `lib.mkEnableOption` / `lib.mkOption`. Never hardcode values in a module's `config` block.

---

## Where to Put Things

| Task | Location |
|------|----------|
| New system-wide package | `modules/general/` or relevant category module |
| New Home Manager package | sub-module under `modules/cli/` or `modules/gui/` |
| New service | new or existing `modules/general/` module |
| New flake input | `flake.nix` → `inputs` block |
| New overlay | `overlays/`, then wire into `nixpkgs/` |
| New profile | `profiles/<n>/` + entry in `flake.nix` `nixosConfigurations` |
| Neovim plugin | `modules/cli/home/nvim/plugins/<category>/` |
| New feature flag | declare in `modules/options.nix`, implement in relevant module |

---

## Build Workflow

```bash
just test        # nix flake check — catch eval errors first
just dry-build   # preview closure diff
just build       # nixos-rebuild switch
just fmt         # format all .nix files
just update      # update all flake inputs
just updatep <input>  # update one input
```

---

## Keeping This Skill and README Up to Date

If you notice that the actual file structure or `modules/options.nix` options differ
from what's documented here, flag it to the user and ask whether to update this skill
before proceeding. Do not silently correct it.

**Before any commit**: update both this skill file and the relevant README(s) to reflect
any structural or behavioral changes made — new modules, removed packages, new feature
flags, new flake inputs, etc. Do not commit without keeping documentation in sync.
