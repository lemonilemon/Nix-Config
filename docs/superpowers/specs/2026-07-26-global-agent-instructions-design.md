# Global agent instructions in nix

Manage one set of cross-tool agent instructions from the repo, and deliver it to
every installed AI coding agent's global instruction path.

Ships as **two independent commits**. Change 1 does not depend on Change 2, and
either can be reverted without touching the other.

## Problem

`~/.claude/CLAUDE.md` is an unmanaged file in `$HOME`. It is not in the repo, not
versioned, not reproducible across hosts, and it reaches only Claude Code. The
other four agents installed by `modules/cli/home/programs/ai.nix` (Codex,
OpenCode, antigravity-cli, kiro-cli) each read their own global instruction file
and currently get nothing.

## Research findings

Paths were confirmed by inspecting the installed binaries, not from memory.

| Tool | Binary | Global instruction path | Confidence |
|------|--------|------------------------|------------|
| Claude Code 2.1.220 | `claude` | `~/.claude/CLAUDE.md` | Confirmed in binary |
| Codex 0.145.0 | `codex` | `~/.codex/AGENTS.md` (`$CODEX_HOME/AGENTS.md`) | Confirmed in binary |
| OpenCode 1.18.5 | `opencode` | `~/.config/opencode/AGENTS.md` | Confirmed in binary |
| kiro-cli 2.13.0 | `kiro-cli` | `~/.kiro/steering/00-global.md` | Inferred, see below |
| antigravity-cli 1.1.7 | `agy` | none; walks up from workspace | Confirmed absent |

Two findings drove design changes:

**antigravity-cli has no global instruction file.** It replaced gemini-cli, which
is no longer installed. `agy` reads the filenames `AGENTS.md` and `GEMINI.md`, but
every `.gemini/` path in its binary is runtime state: `settings.json`, `artifacts`,
`transcript.jsonl`, and the `knowledge/` and `implicit/` directories, which hold
its own agent-written memory rather than user-authored instructions. The existing
`~/.gemini/GEMINI.md` is a 0-byte leftover from gemini-cli dated 2026-01-12,
predating the `antigravity-cli/` directory (2026-06-18). Writing it would be a
no-op, so it is not a target.

**kiro-cli uses a steering directory.** Its binary documents default agent
resources as "global/workspace steering, skills, and project marker files like
AGENTS.md". Global home is `~/.kiro` (per `GlobalPaths` symbols) and the steering
subpath is `.kiro/steering`. The target is therefore a file *inside* a directory,
not a single fixed filename. This is marked inferred: kiro-cli has never been run
on this machine, so `~/.kiro` does not exist and the path could not be confirmed
empirically. Verify after first run.

**The shared path.** No path is read by all agents. The closest equivalent is
`~/AGENTS.md`: agents that resolve `AGENTS.md` by walking up the directory tree
(agy, Codex, kiro-cli) pick it up for any project under `$HOME`. This is the only
way to reach `agy`, and it covers future tree-walking agents without new wiring.

## Change 1: honest `home.cli.programs.enable` gate

Independent bug fix, shipped first.

`modules/cli/home/programs/default.nix` places its imports outside its own
`config = lib.mkIf config.home.cli.programs.enable { ... }` block. Imports are
always evaluated, so `ai.nix`, `atuin.nix`, and `fastfetch.nix` apply
unconditionally. Setting `home.cli.programs.enable = false` today still installs
every AI agent, atuin, and fastfetch.

This is a consistency fix, not a new convention. The two remaining siblings,
`utils.nix` and `yazi.nix`, already open with
`config = lib.mkIf config.home.cli.programs.enable { ... }`. Three modules simply
never adopted it.

Fix: wrap each straggler's own config in the same `lib.mkIf`.

| Module | Currently gated |
|--------|-----------------|
| `utils.nix` | yes, leave alone |
| `yazi.nix` | yes, leave alone |
| `ai.nix` | no, fix |
| `atuin.nix` | no, fix |
| `fastfetch.nix` | no, fix |

`atuin.nix` takes `isWSL` and no `lib`/`config`, so its argument set grows by two.

This must land before Change 2 so the new instruction files inherit a gate that
already works.

## Change 2: wire the instruction files

### Layout

`modules/cli/home/programs/ai.nix` becomes a directory, matching how `nvim/`,
`git/`, and `eww/` already handle modules with adjacent assets:

```
modules/cli/home/programs/ai/
├── default.nix     # former ai.nix, plus file wiring
└── AGENTS.md       # canonical content, single source of truth
```

The canonical file is named `AGENTS.md` because it is the cross-tool convention
and three of five targets expect that exact name. Claude Code's `CLAUDE.md` is the
outlier, absorbed by the symlink.

### Wiring

```nix
let
  agentInstructions = ./AGENTS.md;
in
{
  home.file = {
    "AGENTS.md".source = agentInstructions;
    ".claude/CLAUDE.md".source = agentInstructions;
    ".codex/AGENTS.md".source = agentInstructions;
    ".kiro/steering/00-global.md".source = agentInstructions;
  };
  xdg.configFile."opencode/AGENTS.md".source = agentInstructions;
}
```

All five targets symlink to the same `/nix/store` path, so they cannot drift.
`readlink -f` on any two resolves identically.

OpenCode uses `xdg.configFile` rather than the `instructions` key in
`programs.opencode.settings`, keeping one uniform mechanism across all targets
instead of mixing a config-schema knob with symlinks.

### Content

`~/.claude/CLAUDE.md` moves into the repo verbatim. It already reads as
tool-agnostic ("lemonilemon's agent instructions ... across all scenarios"), so
no rewording is needed for the wider audience.

### Migration

`lib/builders.nix:67` sets `backupFileExtension = "backup"`, so activation moves
the existing `~/.claude/CLAUDE.md` aside to `~/.claude/CLAUDE.md.backup` rather
than aborting. No manual pre-flight step.

## Accepted trade-offs

**The global file becomes read-only.** Store symlinks are not writable, so Claude
Code's `#` shortcut and `/memory` can no longer append to the *global* file. Edits
go through the repo and a rebuild. Per-project memory under
`~/.claude/projects/*/memory/` is unaffected, as is `~/.claude/settings.json`,
which stays unmanaged.

**`~/AGENTS.md` is visible in `$HOME`** and applies to every directory beneath it,
including non-code directories. Accepted as the cost of reaching `agy`.

**`~/.kiro/steering/00-global.md` is a guess** until kiro-cli is run once.

## Verification

Per repo convention, all four must pass:

```
just fmt && just check && just test && just dry-build
```

Then, after the user runs `just build`:

```bash
readlink -f ~/AGENTS.md ~/.claude/CLAUDE.md ~/.codex/AGENTS.md \
  ~/.config/opencode/AGENTS.md ~/.kiro/steering/00-global.md
```

All five must print the same store path. Confirm `~/.claude/CLAUDE.md.backup`
holds the old content.

## Out of scope

- `~/.claude/settings.json`, which stays unmanaged
- Per-host or per-profile instruction variants; the file is identical everywhere
  until a second variant is actually needed
- Project-level `CLAUDE.md` / `AGENTS.md`, which stay with their repos
- `~/.gemini/GEMINI.md`, which no installed tool reads
