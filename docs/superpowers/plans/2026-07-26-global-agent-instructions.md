# Global Agent Instructions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Manage one cross-tool agent instruction file from the repo and symlink it into every installed AI agent's global instruction path.

**Architecture:** A single `AGENTS.md` lives beside the AI module. Home Manager symlinks it to five paths, all resolving to the same `/nix/store` object so they cannot drift. Ships as two independent changesets: first an honest `home.cli.programs.enable` gate (Tasks 1-2), then the file wiring (Tasks 3-6).

**Tech Stack:** Nix flakes, Home Manager, `nix flake check` with eval-time assertions in `tests/nix/host-options.nix`.

**Spec:** `docs/superpowers/specs/2026-07-26-global-agent-instructions-design.md`

---

## Orientation for the implementer

**Testing works differently here.** There is no unit test runner for the Nix side. The test suite is `tests/nix/host-options.nix`, a list of `{ name; actual; expected; }` attrsets evaluated at build time. If any `actual != expected`, the file `throw`s and `nix flake check` fails. Add a case to the `expectations` list to write a test.

**Two gotchas that will waste your time:**

1. `grep` is aliased to ripgrep in this shell. `\|` is not alternation. Use `grep -E` with `|`, or call `/run/current-system/sw/bin/grep`. The repo's README files contain Nerd Font glyphs that make real `grep` report "binary file matches", so add `-a` when grepping them.
2. Never run `just build`. It uses sudo and switches the live system. That is the user's call. `just dry-build` also uses sudo, so use the non-sudo form given in the verification steps below.

**Definition of done for every commit:** `just fmt`, then `just check`, then `just test`, then the dry-build command. All four must pass.

---

## File Structure

| File | Change | Responsibility |
|------|--------|----------------|
| `tests/nix/host-options.nix` | Modify | Add gate assertions (Task 1) and wiring assertions (Task 4) |
| `modules/cli/home/programs/atuin.nix` | Modify | Gate on `home.cli.programs.enable` |
| `modules/cli/home/programs/fastfetch.nix` | Modify | Gate on `home.cli.programs.enable` |
| `modules/cli/home/programs/ai.nix` | Delete | Becomes `ai/default.nix` |
| `modules/cli/home/programs/ai/default.nix` | Create | Former `ai.nix`, gated, plus file wiring |
| `modules/cli/home/programs/ai/AGENTS.md` | Create | Canonical instruction content, single source of truth |
| `modules/cli/home/programs/default.nix` | Modify | Import path `./ai.nix` becomes `./ai` |
| `modules/cli/README.md` | Modify | Structure block and stale `ai.nix` description |

---

# CHANGE 1: honest `home.cli.programs.enable` gate

Independent of Change 2. Ships as its own commit.

## Task 1: Write the failing gate assertions

**Files:**
- Modify: `tests/nix/host-options.nix:1-10` (the `let` block) and the `expectations` list

**Background:** `modules/cli/home/programs/default.nix` puts its `imports` outside its own `config = lib.mkIf config.home.cli.programs.enable { ... }`. Imports always evaluate, so `ai.nix`, `atuin.nix`, and `fastfetch.nix` apply unconditionally. `utils.nix` and `yazi.nix` already gate themselves correctly.

- [ ] **Step 1: Add the `programsOff` bindings to the `let` block**

In `tests/nix/host-options.nix`, immediately after the existing `hosts = { ... };` attrset and before `expectations = [`, insert:

```nix
  # A desktop with the programs flag off. Proves home.cli.programs.enable
  # actually gates modules/cli/home/programs/, rather than only gating
  # default.nix's own config block.
  programsOff =
    (nixosConfigurations.desktop.extendModules {
      modules = [ { home.cli.programs.enable = false; } ];
    }).config.home-manager.users.lemonilemon;

  programsOffPackageNames = map (p: p.pname or p.name or "") programsOff.home.packages;
```

- [ ] **Step 2: Add the assertions**

Append these five entries to the end of the `expectations` list, after the `laptop/eww.wifi.enable` entry and before the closing `];`:

```nix
    # --- home.cli.programs.enable actually gates programs/ ---
    {
      name = "programs off/atuin.enable";
      actual = programsOff.programs.atuin.enable;
      expected = false;
    }
    {
      name = "programs off/fastfetch.enable";
      actual = programsOff.programs.fastfetch.enable;
      expected = false;
    }
    {
      name = "programs off/opencode.enable";
      actual = programsOff.programs.opencode.enable;
      expected = false;
    }
    {
      name = "programs off/no AI agent packages";
      actual = builtins.any (n: lib.hasInfix "claude-code" n) programsOffPackageNames;
      expected = false;
    }
    {
      name = "programs off/no sandbox helpers";
      actual = builtins.any (n: n == "socat") programsOffPackageNames;
      expected = false;
    }
```

- [ ] **Step 3: Run the check to verify it fails**

```bash
just check
```

Expected: FAIL. `host-options assertions failed:` followed by all five names, each reporting `expected false, got true`. If any of the five is already `false`, stop and re-read the module; the defect is not what the plan describes.

Do not commit here. The test and its fix commit together in Task 2, so no commit in history leaves `nix flake check` broken.

---

## Task 2: Gate the three straggler modules

**Files:**
- Modify: `modules/cli/home/programs/ai.nix:1-2` and closing brace
- Modify: `modules/cli/home/programs/atuin.nix:1-5` and closing brace
- Modify: `modules/cli/home/programs/fastfetch.nix:1-7` and closing brace

Follow the shape `utils.nix` and `yazi.nix` already use. Do not reformat anything else; `just fmt` handles indentation at the end.

- [ ] **Step 1: Gate `atuin.nix`**

Replace the header. Current first five lines:

```nix
{
  isWSL,
  ...
}:
{
```

Become:

```nix
{
  lib,
  config,
  isWSL,
  ...
}:
{
  config = lib.mkIf config.home.cli.programs.enable {
```

Then add one closing `};` immediately before the file's final `}`, so the braces balance. Do not hand-fix the indentation of the wrapped body; `just fmt` in Step 4 reindents it.

- [ ] **Step 2: Gate `fastfetch.nix`**

Current header:

```nix
{
  lib,
  pkgs,
  ...
}:
{
  programs.fastfetch = {
```

Becomes:

```nix
{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.cli.programs.enable {
    programs.fastfetch = {
```

Add one closing `};` before the file's final `}`. This is a 284-line file; the body between those markers is wrapped one level deeper, so let `just fmt` fix indentation rather than hand-editing every line.

- [ ] **Step 3: Gate `ai.nix`**

Current header:

```nix
{ pkgs, ... }:
{
  home.packages = (
```

Becomes:

```nix
{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.cli.programs.enable {
    home.packages = (
```

Add one closing `};` before the file's final `}`.

- [ ] **Step 4: Format**

```bash
just fmt
```

Expected: exits 0. If `nixfmt` reports a parse error, the braces do not balance; recount the closers added in Steps 1 to 3.

- [ ] **Step 5: Run the check to verify the assertions now pass**

```bash
just check
```

Expected: PASS, no `host-options assertions failed` output.

- [ ] **Step 6: Confirm the flag still leaves the real hosts intact**

The gate must turn things off only when asked. Verify the desktop is unaffected:

```bash
nix eval --impure --expr '
let f = builtins.getFlake (toString ./.);
    hm = f.nixosConfigurations.desktop.config.home-manager.users.lemonilemon;
in { atuin = hm.programs.atuin.enable; fastfetch = hm.programs.fastfetch.enable; opencode = hm.programs.opencode.enable; }'
```

Expected: `{ atuin = true; fastfetch = true; opencode = true; }`

- [ ] **Step 7: Run the remaining verification**

```bash
just test
nix build --dry-run .#nixosConfigurations.desktop.config.system.build.toplevel
```

Expected: both exit 0.

- [ ] **Step 8: Commit**

```bash
git add modules/cli/home/programs/ai.nix \
        modules/cli/home/programs/atuin.nix \
        modules/cli/home/programs/fastfetch.nix \
        tests/nix/host-options.nix
git commit -m "fix(cli): gate ai, atuin and fastfetch on home.cli.programs.enable

These three modules set config at the top level, but programs/default.nix
puts its imports outside its own lib.mkIf. Imports always evaluate, so
home.cli.programs.enable = false still installed every AI agent, atuin and
fastfetch. utils.nix and yazi.nix already gated themselves; this brings the
remaining three in line."
```

**Change 1 is complete.** It can ship or be reverted without touching Change 2.

---

# CHANGE 2: wire the instruction files

## Task 3: Restructure the AI module into a directory

Pure move, no behaviour change. Keeping it separate from Task 4 keeps the wiring diff readable.

**Files:**
- Delete: `modules/cli/home/programs/ai.nix`
- Create: `modules/cli/home/programs/ai/default.nix`
- Modify: `modules/cli/home/programs/default.nix:9`

- [ ] **Step 1: Move the file, preserving history**

```bash
mkdir -p modules/cli/home/programs/ai
git mv modules/cli/home/programs/ai.nix modules/cli/home/programs/ai/default.nix
```

- [ ] **Step 2: Update the import**

In `modules/cli/home/programs/default.nix`, the imports list currently reads:

```nix
  imports = [
    ./fastfetch.nix
    ./atuin.nix
    ./ai.nix
    ./utils.nix
    ./yazi.nix
  ];
```

Change `./ai.nix` to `./ai`:

```nix
  imports = [
    ./fastfetch.nix
    ./atuin.nix
    ./ai
    ./utils.nix
    ./yazi.nix
  ];
```

- [ ] **Step 3: Verify the move changed nothing**

```bash
just check && just test
```

Expected: both pass, including the five gate assertions from Task 1.

- [ ] **Step 4: Commit the move**

```bash
git add -A modules/cli/home/programs/
git commit -m "refactor(cli): make the ai module a directory

Prepares for an adjacent AGENTS.md asset, matching how nvim/, git/ and
eww/ already pair a default.nix with its data files."
```

---

## Task 4: Write the failing wiring assertions

**Files:**
- Modify: `tests/nix/host-options.nix` (the `let` block and `expectations` list)

- [ ] **Step 1: Add the `agentFiles` bindings to the `let` block**

After the `programsOffPackageNames` binding added in Task 1, insert:

```nix
  # Every agent instruction target must resolve to one store object, so the
  # tools cannot drift apart. See docs/superpowers/specs/2026-07-26-*.md for
  # how each path was confirmed against the installed binaries.
  desktopHome = hosts.desktop.home-manager.users.lemonilemon;

  agentInstructionSources = [
    desktopHome.home.file."AGENTS.md".source
    desktopHome.home.file.".claude/CLAUDE.md".source
    desktopHome.home.file.".codex/AGENTS.md".source
    desktopHome.home.file.".kiro/steering/00-global.md".source
    desktopHome.xdg.configFile."opencode/AGENTS.md".source
  ];
```

- [ ] **Step 2: Add the assertions**

Append to the end of the `expectations` list:

```nix
    # --- global agent instructions ---
    {
      name = "agent instructions/all five targets share one store path";
      actual = builtins.all (
        s: builtins.toString s == builtins.toString (builtins.head agentInstructionSources)
      ) agentInstructionSources;
      expected = true;
    }
    {
      name = "agent instructions/content reaches the store";
      actual = lib.hasInfix "lemonilemon's agent instructions" (
        builtins.readFile (builtins.head agentInstructionSources)
      );
      expected = true;
    }
    {
      # The flag gates the instruction files too, not just the packages.
      name = "programs off/no agent instruction files";
      actual = programsOff.home.file ? "AGENTS.md";
      expected = false;
    }
```

- [ ] **Step 3: Run the check to verify it fails**

```bash
just check
```

Expected: FAIL. The first two assertions error while evaluating, because `home.file."AGENTS.md"` does not exist yet. A missing-attribute error is the correct red state here; it still proves the wiring is absent. The third assertion reports `expected false, got false` only once wiring exists, so it is inert for now.

Do not commit a failing check. Move straight to Task 5.

---

## Task 5: Add the content and the wiring

**Files:**
- Create: `modules/cli/home/programs/ai/AGENTS.md`
- Modify: `modules/cli/home/programs/ai/default.nix`

- [ ] **Step 1: Create the canonical instruction file**

Write `modules/cli/home/programs/ai/AGENTS.md` with exactly the current content of `~/.claude/CLAUDE.md`:

```markdown
# lemonilemon's agent instructions
These are common instructions for lemonilemon's agents across all scenarios.

## General Guidelines
- DO NOT overuse the em dash (—) in your responses. Only use it when it deserves.
- When writing commit messages, NEVER auto-add your agent name as co-author.
- When making a technical decision, DO NOT give much weight about the development cost. Focus on the long-term benefits and maintainability of the solution.
- When doing end-to-end testing, please be picky about the UI and UX even for pixel-perfectness.
- Design specs and implementation plans are scaffolding: committed while the work is in flight, deleted once the feature ships.
```

- [ ] **Step 2: Verify it matches the live file byte for byte**

```bash
diff ~/.claude/CLAUDE.md modules/cli/home/programs/ai/AGENTS.md && echo IDENTICAL
```

Expected: `IDENTICAL`. Any diff means the content was retyped rather than copied; fix it before continuing.

- [ ] **Step 3: Add the wiring to `ai/default.nix`**

Introduce a `let` binding above the module body. The file currently opens:

```nix
{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.cli.programs.enable {
```

Becomes:

```nix
{
  lib,
  config,
  pkgs,
  ...
}:
let
  # One file, five symlinks, all resolving to the same store object. Paths were
  # confirmed against the installed binaries; see the spec for the evidence and
  # for why antigravity-cli gets ~/AGENTS.md rather than a dedicated path.
  agentInstructions = ./AGENTS.md;
in
{
  config = lib.mkIf config.home.cli.programs.enable {
```

Then, inside the `config = lib.mkIf ... {` block, alongside the existing `home.packages` and `programs.opencode` attributes, add:

```nix
    # Tree-walking agents (antigravity-cli's `agy`, codex, kiro-cli) pick this
    # up for any project under $HOME. It is the only path that reaches `agy`,
    # which has no global instruction file of its own.
    home.file = {
      "AGENTS.md".source = agentInstructions;
      ".claude/CLAUDE.md".source = agentInstructions;
      ".codex/AGENTS.md".source = agentInstructions;
      # Inferred path: kiro-cli has never run here, so ~/.kiro does not exist.
      # Its binary documents "global/workspace steering" as a default resource
      # and its global home is ~/.kiro. Re-verify after the first kiro-cli run.
      ".kiro/steering/00-global.md".source = agentInstructions;
    };

    xdg.configFile."opencode/AGENTS.md".source = agentInstructions;
```

- [ ] **Step 4: Format**

```bash
just fmt
```

Expected: exits 0.

- [ ] **Step 5: Run the check to verify the assertions pass**

```bash
just check
```

Expected: PASS. All three Task 4 assertions plus the five from Task 1 are green.

- [ ] **Step 6: Confirm all five targets resolve identically in the evaluated config**

```bash
nix eval --impure --expr '
let f = builtins.getFlake (toString ./.);
    hm = f.nixosConfigurations.desktop.config.home-manager.users.lemonilemon;
in builtins.map toString [
     hm.home.file."AGENTS.md".source
     hm.home.file.".claude/CLAUDE.md".source
     hm.home.file.".codex/AGENTS.md".source
     hm.home.file.".kiro/steering/00-global.md".source
     hm.xdg.configFile."opencode/AGENTS.md".source
   ]'
```

Expected: a list of five identical `/nix/store/...-AGENTS.md` paths.

- [ ] **Step 7: Run the remaining verification**

```bash
just test
nix build --dry-run .#nixosConfigurations.desktop.config.system.build.toplevel
```

Expected: both exit 0.

---

## Task 6: Update the READMEs, then commit

The repo requires READMEs to be in sync before any commit.

**Files:**
- Modify: `modules/cli/README.md:27` and `modules/cli/README.md:211`

- [ ] **Step 1: Update the structure block**

At `modules/cli/README.md:27`, the line reads:

```
      programs/       # CLI utilities
```

Change to:

```
      programs/       # CLI utilities
         ai/         # AI agents + the shared AGENTS.md instruction file
```

Match the surrounding indentation exactly; the block uses spaces, not tabs.

- [ ] **Step 2: Replace the stale AI description**

`modules/cli/README.md:211` currently reads:

```
- **ai.nix**: AI tools — `claude-code`, `gemini-cli`, and `opencode` with a custom multi-agent setup (team-lead, product-manager, developer, code-reviewer). Also installs `socat` and `bubblewrap` for sandboxing.
```

This is stale: `gemini-cli` is no longer installed. Replace with:

```
- **ai/**: AI coding agents — `claude-code`, `codex`, `antigravity-cli` (the `agy` binary, which replaced `gemini-cli`), `kiro-cli`, `skills`, `ccusage`, and `opencode` with a custom multi-agent setup (team-lead, product-manager, developer, code-reviewer). Also installs `socat` and `bubblewrap` for sandboxing.
- **ai/AGENTS.md**: one set of global agent instructions, symlinked to `~/AGENTS.md`, `~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md`, `~/.config/opencode/AGENTS.md` and `~/.kiro/steering/00-global.md`. All five resolve to the same store path, so they cannot drift. Edit this file and rebuild; the global file is read-only, so agents cannot append to it in place.
```

- [ ] **Step 3: Re-run the full verification**

```bash
just fmt && just check && just test
nix build --dry-run .#nixosConfigurations.desktop.config.system.build.toplevel
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add modules/cli/home/programs/ai/ tests/nix/host-options.nix modules/cli/README.md
git commit -m "feat(cli): manage global agent instructions from nix

One AGENTS.md beside the ai module, symlinked to all five global agent
instruction paths so they resolve to a single store object and cannot
drift. Replaces the unmanaged ~/.claude/CLAUDE.md.

antigravity-cli has no global instruction path of its own, so ~/AGENTS.md
covers it and any other agent that walks up from the workspace. The
kiro-cli steering path is inferred and needs re-checking after its first
run. ~/.gemini/GEMINI.md is deliberately not written; no installed tool
reads it."
```

---

## Task 7: Hand off to the user for activation

`just build` uses sudo and switches the live system. Do not run it.

- [ ] **Step 1: Report the post-activation check to the user**

Tell them to run `just build`, then verify:

```bash
readlink -f ~/AGENTS.md ~/.claude/CLAUDE.md ~/.codex/AGENTS.md \
  ~/.config/opencode/AGENTS.md ~/.kiro/steering/00-global.md
```

Expected: five identical store paths.

- [ ] **Step 2: Flag the backup file**

`lib/builders.nix:67` sets `backupFileExtension = "backup"`, so activation moves the old file aside automatically. Confirm with:

```bash
cat ~/.claude/CLAUDE.md.backup
```

Expected: the previous unmanaged content. Once the user confirms the symlink works, this file can be deleted.

- [ ] **Step 3: Note the kiro-cli caveat**

`~/.kiro/steering/00-global.md` is the one inferred path. After the user first runs `kiro-cli`, check that it actually reads the file. If not, the fix is a global agent config under `~/.kiro/agents/` with a `resources` entry pointing at the file, rather than a steering drop-in.

- [ ] **Step 4: Delete the scaffolding**

Per the user's standing convention, specs and plans are deleted once the feature ships. After the user confirms activation works:

```bash
git rm docs/superpowers/specs/2026-07-26-global-agent-instructions-design.md \
       docs/superpowers/plans/2026-07-26-global-agent-instructions.md
git commit -m "chore(docs): drop shipped agent instruction spec and plan"
```
