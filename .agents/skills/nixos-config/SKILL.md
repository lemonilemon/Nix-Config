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
- Profiles, shared fragments (`base.nix`, `boot.nix`, …), new-profile walkthrough,
  `NIXHOST` convention → [profiles/README.md](../../profiles/README.md)

Then, if the file tree isn't already visible:
```bash
ls && cat flake.nix
```

## Source of Truth — Read the Code, Not a Copy

This skill deliberately does not duplicate option tables or input lists; hand-maintained
copies drift. The authoritative files are small — read them directly:

| Question | Read this |
|----------|-----------|
| What feature flags exist? Defaults? | `modules/options.nix` (top-level) + `modules/{cli,gui,desktop,general}/options.nix` |
| What flake inputs exist? | `flake.nix` `inputs` block |
| How are systems/homes assembled? | `lib/builders.nix` — concepts in [`references/builders.md`](references/builders.md) |
| How are overlays applied? | `nixpkgs/overlays.nix` (wires `overlays/`) |
| What does a category module contain? | `modules/<category>/README.md` |

For **upstream** NixOS / Home Manager / nixvim option names and package attributes, use the
`mcp-nixos` tools (`nix`, `nix_versions`) if available rather than guessing from memory —
option names change between releases.

## Modular Philosophy

Options are declared centrally in `modules/options.nix` and per-category `options.nix` files;
modules implement them via `lib.mkIf`. Profiles only set option values — they do not contain
inline NixOS/HM config.

Two conventions to preserve when adding options:

1. **Cascading defaults**: a sub-option defaults to its parent's value
   (e.g. `home.cli.git.enable` defaults to `config.home.cli.enable`), so toggling a parent
   flips the whole subtree unless a profile overrides a leaf. New sub-options must follow
   this pattern or they break the cascade.
2. **HM options mirror NixOS**: Home Manager `options.nix` files use `helpers.mkHomeOpt`
   so the HM value follows `osConfig` when running inside NixOS and falls back to `default`
   standalone. See [`references/builders.md`](references/builders.md).

**Adding options that don't exist yet**: declare them explicitly with
`lib.mkOption` / `helpers.mkHomeOpt` in the relevant `options.nix`. Never hardcode values in
a module's `config` block.

## Where to Put Things

| Task | Location |
|------|----------|
| New system-wide package | `modules/general/` or relevant category module |
| New Home Manager package | sub-module under `modules/cli/` or `modules/gui/` |
| New service | new or existing `modules/general/` module |
| New flake input | `flake.nix` → `inputs` block (patterns below) |
| New overlay | `overlays/`, then wire into `nixpkgs/overlays.nix` |
| New profile | `profiles/<name>/` + entry in `flake.nix` `nixosConfigurations` |
| Neovim plugin | `modules/cli/home/nvim/plugins/<category>/` |
| New feature flag | declare in the relevant `options.nix`, implement in the module |

## Flake Input Patterns

Standard input (add `follows` only if the input accepts it):
```nix
my-input = {
  url = "github:owner/repo";
  inputs.nixpkgs.follows = "nixpkgs";
};
```

Non-flake source (e.g. Neovim plugin pinned to a commit):
```nix
my-plugin = {
  url = "github:owner/repo/COMMIT_HASH";
  flake = false;
};
```

All inputs reach modules as `inputs.<name>` via `specialArgs` (wired by `mkSystem` / `mkHome`).

Repo-specific gotcha: the `llm-agents` input (AI coding agents: claude-code, gemini-cli, …)
is exposed through an overlay in `nixpkgs/overlays.nix`; its packages live under
`pkgs.llm-agents.*`, not as a direct flake-output reference.

## Build & Verify Workflow

Recipes live in `Justfile`. The NixOS recipes target the host named by the **`NIXHOST`
environment variable** (`nixos-rebuild ... --flake .#$NIXHOST`) — if a build fails with an
empty/unknown attribute, check that first.

```bash
just test        # nix eval of the current host config (--show-trace) — catch eval errors first
just dry-build   # preview closure diff without switching
just build       # nixos-rebuild switch
just fmt         # nix fmt — format all .nix files
just update      # update all flake inputs
just updatep <input>   # update one input
just changehost <host> # switch to a different host's config
just gc          # nix-collect-garbage -d
just push        # build toplevel and push closure to cachix
```

**Definition of done for any config change**: run `just fmt`, then `just test`, then
`just dry-build`. All three must pass before presenting the change. Do not run
`just build` yourself — it uses sudo and switches the live system; leave that to the user.

## Keeping This Skill and READMEs Up to Date

Option tables and input lists live only in the code now, so routine changes need no skill
edits. Update this skill only when a **convention** changes (directory layout, builder
signatures, the cascade/mirror patterns, Justfile recipes).

If the repo's actual conventions differ from what's described here, flag it to the user and
ask whether to update this skill before proceeding — do not silently correct it.

**Before any commit**:
- Update the relevant `README.md`(s) to reflect structural or behavioral changes — new
  modules, removed packages, new feature flags, new flake inputs. Do not commit without
  keeping the READMEs in sync.
- Run `just fmt` — `nixfmt` is enforced as a pre-commit hook (wired via
  `checks.pre-commit-check` in `flake.nix`), so unformatted files fail the commit.
- Use conventional-commit style messages (`feat:`, `fix:`, `refactor:`, …), matching the
  existing history.
